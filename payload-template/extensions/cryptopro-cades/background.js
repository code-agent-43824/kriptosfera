var g_content_ports = {};
var g_native_ports = {};
var g_message_collector = {};
var LOG_LEVEL_DEBUG = 4;
var LOG_LEVEL_INFO = 2;
var LOG_LEVEL_ERROR = 1;
var current_log_level = LOG_LEVEL_ERROR;
var g_user_approved_sites = {};

function cpcsp_console_log(level, msg){
    if (level <= current_log_level){
        if (level == LOG_LEVEL_DEBUG)
            console.log("DEBUG: %s", msg);
        if (level == LOG_LEVEL_INFO)
            console.info("INFO: %s", msg);
        if (level == LOG_LEVEL_ERROR)
            console.error("ERROR: %s", msg);
        return;
    }
}

function isValidApprovedSite(site) {
    if (g_user_approved_sites[site])
        return Date.now() < g_user_approved_sites[site] + 24 * 60 * 60 * 1000;
    return false;
}

function set_log_level(level){
    current_log_level = level;
    if (current_log_level == LOG_LEVEL_DEBUG)
        cpcsp_console_log(LOG_LEVEL_INFO, "background.js: log_level = DEBUG");
    if (current_log_level == LOG_LEVEL_INFO)
        cpcsp_console_log(LOG_LEVEL_INFO, "background.js: log_level = INFO");
    if (current_log_level == LOG_LEVEL_ERROR)
        cpcsp_console_log(LOG_LEVEL_INFO, "background.js: log_level = ERROR");
}

function sendNativeMessage(msg) {
    try {
        g_native_ports[msg.tabid].postMessage(msg);
        cpcsp_console_log(LOG_LEVEL_DEBUG, "background.js: Sent native message:" + JSON.stringify(msg));
    }
    catch(err){
        cpcsp_console_log(LOG_LEVEL_ERROR, "background.js: Exception on sending NativeMessage " + err.message);
        g_native_ports[msg.tabid] = null;
        var err_message = {tabid: msg.tabid, data: {requestid: msg.data.requestid,
                          message: "Error sending message to Native Host"},
                          type: "error"};
        g_content_ports[msg.tabid].postMessage(err_message);
    }
}

var isFireFox = navigator.userAgent.match(/Firefox/i);
var isEdge = navigator.userAgent.match(/Edge/i);
if(!isFireFox && !isEdge){
    browserInstance = chrome;
}else {
    browserInstance = browser;
}

function onNativeMessage(message) {
    try {
        if(isEdge) {
            message = JSON.parse(message);
        }
        cpcsp_console_log(LOG_LEVEL_DEBUG, "background.js: Received native message:" + JSON.stringify(message));
        if (message.partial)
        {
            if(message.partial == 1)
            {
                g_message_collector[message.tabid + ":" + message.requestid] = message.part;
                cpcsp_console_log(LOG_LEVEL_DEBUG, "background.js: Received native message begin");
                if (isEdge) {
                    partial_messsage = {tabid: message.tabid, data: {requestid: message.requestid, type: "get_part", last_part: message.partial}};
                    sendNativeMessage(partial_messsage);
                }
                return;
            }
            if(message.partial > 1)
            {
                g_message_collector[message.tabid + ":" + message.requestid] += message.part;
                cpcsp_console_log(LOG_LEVEL_DEBUG, "background.js: Received native message next part");
                if (isEdge) {
                    partial_messsage = {tabid: message.tabid, data: {requestid: message.requestid, type: "get_part", last_part: message.partial}};
                    sendNativeMessage(partial_messsage);
                }
                return;
            }
            if(message.partial == -1)
            {
                g_message_collector[message.tabid + ":" + message.requestid] += message.part;
                cpcsp_console_log(LOG_LEVEL_DEBUG, "background.js: Received native message end");
                g_content_ports[message.tabid].postMessage(JSON.parse(g_message_collector[message.tabid + ":" + message.requestid]));
                g_message_collector[message.tabid + ":" + message.requestid] = "";
                return;
            }
        }
        if(message.error)
        {
            cpcsp_console_log(LOG_LEVEL_ERROR, "background.js: Received fatal error message from Native Host: " + message.error);
            return;
        }
        if (message.data && message.data.type === "approved_site")
        {
            var site;
            if (!message.data.value.indexOf("add_approved_site:")) {
                site = message.data.value.substring("add_approved_site: ".length);
                g_user_approved_sites[site] = Date.now();
            }
            else if (!message.data.value.indexOf("is_approved_site:")) {
                site = message.data.value.substring("is_approved_site: ".length);
            }
            else {
                cpcsp_console_log(LOG_LEVEL_ERROR, "background.js: Received incorrect approved_site native message:" + JSON.stringify(message));
                return;
            }

            var isApproved = isValidApprovedSite(site);
            args = new Array();
            arg = { type: typeof isApproved, value: isApproved };
            args.push(arg);
            message.data.params = args;
            g_native_ports[message.tabid].postMessage(message);
            cpcsp_console_log(LOG_LEVEL_DEBUG, "background.js: Sent native message:" + JSON.stringify(message));
            return;
        }
        g_content_ports[message.tabid].postMessage(message);
    }
    catch(err){
        cpcsp_console_log(LOG_LEVEL_ERROR, "background.js: Exception on sending message to content page " + err.message);
        g_content_ports[message.tabid] = null;
    }
}

function connect(_content_port) {
    var hostName = "ru.cryptopro.nmcades";
    cpcsp_console_log(LOG_LEVEL_INFO, "background.js: Connecting to native messaging host" + hostName);
    g_native_ports[_content_port.name] = browserInstance.runtime.connectNative(hostName);
    g_native_ports[_content_port.name].onMessage.addListener(onNativeMessage);
    g_native_ports[_content_port.name].onDisconnect.addListener(function () {
        cpcsp_console_log(LOG_LEVEL_INFO, "background.js: Disconnect Event: " + " tabid " + _content_port.name);
        g_native_ports[_content_port.name] = null;
        if(g_content_ports[_content_port.name])
            g_content_ports[_content_port.name].disconnect();
        g_content_ports[_content_port.name] = null;
        browserInstance.pageAction.setIcon({tabId: _content_port.sender.tab.id, path:"icons/plugin_load_error.png"});
        browserInstance.pageAction.setPopup({tabId: _content_port.sender.tab.id, popup:"error_on_plugin_load/error_on_plugin_load.html"});
    });
    return true;
}

browserInstance.runtime.onConnect.addListener(function(_content_port) {
        cpcsp_console_log(LOG_LEVEL_INFO, "background.js: Connected from tabid:" + _content_port.name);
        g_content_ports[_content_port.name] = _content_port;

        _content_port.onMessage.addListener(function(msg) {
            sendNativeMessage(msg);
        });
        _content_port.onDisconnect.addListener(function() {
            g_content_ports[_content_port.name] = null;
            if(g_native_ports[_content_port.name])
                g_native_ports[_content_port.name].disconnect();
            g_native_ports[_content_port.name] = null;
        });
        if(!g_native_ports[_content_port.name])
            connect(_content_port);
        browserInstance.pageAction.show(_content_port.sender.tab.id);
        browserInstance.pageAction.setPopup({tabId: _content_port.sender.tab.id, popup:"plugin_load_ok/plugin_load_ok.html"});
});

browserInstance.runtime.onMessage.addListener(function (request, sender){
    if (request == "set_log_level=debug"){
        set_log_level(LOG_LEVEL_DEBUG);
        return;
    }else if (request == "set_log_level=info"){
        set_log_level(LOG_LEVEL_INFO);
        return;
    }else if (request == "set_log_level=error"){
        set_log_level(LOG_LEVEL_ERROR);
        return;
    }
});

if (browserInstance.declarativeContent) {
    browserInstance.declarativeContent.onPageChanged.removeRules(undefined, function() {
      browserInstance.declarativeContent.onPageChanged.addRules([{
        conditions: [new chrome.declarativeContent.PageStateMatcher()],
            actions: [new chrome.declarativeContent.ShowPageAction()]
      }]);
    });
}

browserInstance.tabs.onUpdated.addListener(function(tabId, changeInfo, tab) {
    if (changeInfo.url && browserInstance.declarativeContent) {
        chrome.pageAction.show(tabId);
    }
});
