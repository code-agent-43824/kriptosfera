;( function () {

    //порт для канала в background.js
    let g_bg_port = null;
    let g_return_window = null;
    const LOG_LEVEL_DEBUG = 4;
    const LOG_LEVEL_INFO = 2;
    const LOG_LEVEL_ERROR = 1;
    let current_log_level = LOG_LEVEL_ERROR;
    let EnableInternalCSP;
    let isFireFox = false;
    let isEdge = false;
    let isSafari = false;
    let isYandex = false;
    let browserInstance;
    let bFirstCall = true;

    function check_browser() {
        var ua= navigator.userAgent, tem, M= ua.match(/(opera|yabrowser|chrome|safari|firefox|msie|trident(?=\/))\/?\s*(\d+)/i) || [];
        if(/trident/i.test(M[1])){
            tem =  /\brv[ :]+(\d+)/g.exec(ua) || [];
            return { name:'IE', version:(tem[1] || '')};
        }
        if(M[1] === 'Chrome'){
            tem = ua.match(/\b(OPR|Edg|YaBrowser)\/(\d+)/);
            if (tem != null)
                return { name: tem[1].replace('OPR', 'Opera'), version: tem[2] };
        }
        M= M[2]? [M[1], M[2]]: [navigator.appName, navigator.appVersion, '-?'];
        if ((tem = ua.match(/version\/(\d+)/i)) != null)
            M.splice(1, 1, tem[1]);
        return {name:M[0],version:M[1]};
    }
    var browserSpecs = check_browser();

    if (browserSpecs.name === 'Edg') {
        isEdge = true;
        browserInstance = chrome;
    }
    if (browserSpecs.name === 'Opera' ||
        browserSpecs.name === 'Chrome') {
        browserInstance = chrome;
    }
    if (browserSpecs.name === 'YaBrowser') {
        isYandex = true;
        browserInstance = chrome;
    }
    if (browserSpecs.name === 'Firefox') {
        isFireFox = true;
        browserInstance = browser;
    }
    if (browserSpecs.name === 'Safari') {
        isSafari = true;
    }

    function yandexConnect() {
        window.addEventListener("message", function (event) {
            if (event.source !== window)
                return;
            if (event.data === "cadesplugin_echo_request") {
                var answer = "cadesplugin_loaded";
                window.postMessage(answer, "*");
                return;
            }
            if (typeof (event.data.destination) === "undefined" || event.data.destination !== "nmcades") {
                return;
            }
            g_return_window = event.source;
            try {
                if (!g_bg_port) {
                    connect(g_tabid);
                }
                g_bg_port.postMessage({ tabid: g_tabid, data: event.data });
            } catch (e) {
                cpcsp_console_log("Error connect to extension when sending request");
                window.postMessage({ tabid: g_tabid, data: { type: "error", requestid: event.data.requestid, message: "Lost connection to extension" } }, "*");
            }
            cpcsp_console_log(LOG_LEVEL_DEBUG, "content.js: Sent message to background:" + JSON.stringify({ tabid: g_tabid, data: event.data }));
        }, false);
        if (bFirstCall) {
            bFirstCall = false;
            window.postMessage("cadesplugin_echo_request", "*");
        }
    }

    var operaExtensionID = "epebfcehmdedogndhlcacafjaacknbcm";
    function operaExtensionLoaded() {
        var extensionID = chrome.runtime.id;
        if (extensionID === operaExtensionID) {
            yandexConnect();
        }
    }
    function operaExtensionFailed() {
        yandexConnect();
    }
    function loadSingleExtension() {
        try {
            if (!bFirstCall)
                return;
            var fileref = document.createElement('script');
            fileref.setAttribute("type", "text/javascript");
            fileref.setAttribute("src", "chrome-extension://" + operaExtensionID + "/nmcades_plugin_api.js");

            fileref.onerror = operaExtensionFailed;
            fileref.onload = operaExtensionLoaded;
            document.getElementsByTagName("head")[0].appendChild(fileref);
        }
        catch (err) {
            cpcsp_console_log(LOG_LEVEL_ERROR, "loadSingleExtension failed. Error: " + err);
        }
    }

    function cadesplugin_not_exists() {
        return !!window.cadesplgin;
    }
//    if (isSafari && !window.cadesplugin) {
//        return;
//    }

    //generate random tabid
    function gen_tabid () {
        function s4() {
            return Math.floor((1 + Math.random()) * 0x10000)
                .toString(16)
                .substring(1);
        }
        return function() {
            return s4() + s4() + '-' + s4() + '-' + s4() + '-' +
                s4() + '-' + s4() + s4() + s4();
        };
    }

    let g_tabid = gen_tabid()();
    function cpcsp_console_log(level, msg){
        if (level <= current_log_level){
            if (level === LOG_LEVEL_DEBUG)
                console.log("DEBUG: %s", msg);
            if (level === LOG_LEVEL_INFO)
                console.info("INFO: %s", msg);
            if (level === LOG_LEVEL_ERROR)
                console.error("ERROR: %s", msg);
        }
    }

    function set_log_level(level){
        current_log_level = level;
        if (current_log_level === LOG_LEVEL_DEBUG)
            cpcsp_console_log(LOG_LEVEL_INFO, "content.js: log_level = DEBUG");
        if (current_log_level === LOG_LEVEL_INFO)
            cpcsp_console_log(LOG_LEVEL_INFO, "content.js: log_level = INFO");
        if (current_log_level === LOG_LEVEL_ERROR)
            cpcsp_console_log(LOG_LEVEL_INFO, "content.js: log_level = ERROR");
    }

    // Установить флаг использования mini-csp
    window.addEventListener("message", function(event) {
        if (event.source !== window)
            return;
        if(event.data === "EnableInternalCSP=true")
        {
            EnableInternalCSP = true;
            cpcsp_console_log(LOG_LEVEL_INFO, "content.js: EnableInternalCSP=true");
        } else if(event.data === "EnableInternalCSP=false")
        {
            EnableInternalCSP = false;
            cpcsp_console_log(LOG_LEVEL_INFO, "content.js: EnableInternalCSP=false");
        }
    }, false);

    function bg_on_message (msg) {
        if(msg.data.type === "result" || msg.data.type === "error") {
            cpcsp_console_log(LOG_LEVEL_DEBUG, "content.js: Sent message to nmcades_plugin:" + JSON.stringify(msg));
            window.postMessage( msg, "*");
            return;
        }
        if(msg.data.type === "callback") {
            let result;
            if(msg.data.value === "result = window.document.URL"){
                result = window.document.URL;
            } else if(msg.data.value === "result = cadesplugin.EnableInternalCSP"){
                if (typeof EnableInternalCSP === "undefined") {
                    result = false;
                } else result = EnableInternalCSP;
            } else if(msg.data.object !== undefined){
                window.postMessage(msg.data, "*");
            } else {
                result = "Internal error on content.js callback call";
                cpcsp_console_log(LOG_LEVEL_ERROR, "content.js: Internal error on content.js callback call " + JSON.stringify(msg.data.value));
            }
            msg.data.type = "result";
            args = new Array();
            arg = {type: typeof result, value: result};
            args.push(arg);
            msg.data.params = args;
            cpcsp_console_log(LOG_LEVEL_DEBUG, "content.js: Sent message to background:" + JSON.stringify(msg));
            if(!isSafari) {
                g_bg_port.postMessage(msg);
            } else {
                safari.extension.dispatchMessage("messageToExtension", {message: JSON.stringify(msg)});
            }
        }
    }

    function connect(){
        if (!isSafari) {
            g_bg_port = browserInstance.runtime.connect({name: g_tabid});
            g_bg_port.onMessage.addListener(function (msg) {
                bg_on_message(msg);
            });
            g_bg_port.onDisconnect.addListener(function() {
                g_bg_port = null;
            });
        } else {
            if( cadesplugin_not_exists())
                return;
            g_bg_port = 1;
            safari.self.addEventListener("message", function (event) {
                let data = JSON.parse(event.message.data);
                bg_on_message(data);
            });
            setInterval(() => safari.extension.dispatchMessage("messageToExtension",{message: "Ping"}), 25000);
            //Вызывается когда происходит переход на странице на другую страницу или релоад.
            window.addEventListener("pagehide", function(event) {
                safari.extension.dispatchMessage("messageToExtension", {message: JSON.stringify({tabid: g_tabid, data: {type: "Page reloaded"}})});
            });
        }
    }

    //Установить уровень логов
    window.addEventListener("message", function(event) {
        if (event.source !== window || isSafari)
            return;
        if(event.data === "set_log_level=debug")
        {
            set_log_level(LOG_LEVEL_DEBUG);
            browserInstance.runtime.sendMessage("set_log_level=debug");
        } else if(event.data === "set_log_level=info")
        {
            set_log_level(LOG_LEVEL_INFO);
            browserInstance.runtime.sendMessage("set_log_level=info");
        } else if(event.data === "set_log_level=error")
        {
            set_log_level(LOG_LEVEL_ERROR);
            browserInstance.runtime.sendMessage("set_log_level=error");
        }
    }, false);

    // CADES-2138
    function singlePostMessage(answer) {
        if (isYandex) {
            loadSingleExtension();
        } else {
            window.postMessage(answer, "*");
        }
    }

    //EventListner для обработки ситуации когда расширение загрузилось раньше скрипта на странице
    // в такой ситуации начальное сообщение будет пропущенно и нам нужно ждать "cadesplugin_echo_request"
    window.addEventListener("message", function (event) {
        if (event.source !== window || isYandex)
            return;
        if (event.data === "cadesplugin_echo_request")
        {
            var answer = "cadesplugin_loaded";
            if (isFireFox) {
                answer += "url:";
                answer += browser.extension.getURL("nmcades_plugin_api.js");
            }
            if (isSafari) {
                answer += "url:";
                answer += safari.extension.baseURI + "nmcades_plugin_api.js";
            }
            if (isEdge) {
                answer += "url:";
                answer += chrome.extension.getURL("nmcades_plugin_api.js");
            }

            singlePostMessage(answer);
            return;
        }
        if (typeof (event.data.destination) === "undefined" || event.data.destination !== "nmcades") {
            return;
        }
        g_return_window = event.source;
        try {
            if (!g_bg_port)
                connect(g_tabid);
            if (!isSafari) {
                g_bg_port.postMessage({ tabid: g_tabid, data: event.data });
            } else {
                if (cadesplugin_not_exists())
                    return;
                safari.extension.dispatchMessage("messageToExtension", { message: JSON.stringify({ tabid: g_tabid, data: event.data }) });
            }
        } catch (e) {
            cpcsp_console_log("Error connect to extension when sending request");
            window.postMessage({ tabid: g_tabid, data: { type: "error", requestid: event.data.requestid, message: "Lost connection to extension" } }, "*");
        }
        cpcsp_console_log(LOG_LEVEL_DEBUG, "content.js: Sent message to background:" + JSON.stringify({ tabid: g_tabid, data: event.data }));
    }, false);

    let answer = "cadesplugin_loaded";
    if (isFireFox || isEdge) {
        answer += "url:";
        answer += browserInstance.extension.getURL("nmcades_plugin_api.js");
    } else if (isSafari) {
        answer += "url:";
        answer += safari.extension.baseURI + "nmcades_plugin_api.js";
    }

    singlePostMessage(answer);
    window.postMessage("EnableInternalCSP_request", "*");

    cpcsp_console_log(LOG_LEVEL_INFO, "content.js: Cadesplugin loaded");
}());
