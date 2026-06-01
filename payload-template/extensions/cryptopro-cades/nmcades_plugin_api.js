;( function () {
    //already loaded
    if(window.cpcsp_chrome_nmcades)
        return;

    let cpcsp_chrome_nmcades = {};
    var callbackFuncs = [];

    function is_cadesplugin_defined() {
        return typeof cadesplugin !== 'undefined';
    }

    function Print2Digit(digit)
    {
        return (digit<10) ? "0"+digit : digit;
    }

    function DateToUTCStr(d)
    {
        let ret = d.getUTCFullYear() + "-";
        ret = ret + Print2Digit(d.getUTCMonth() + 1) ;
        ret = ret + "-";
        ret = ret + Print2Digit(d.getUTCDate()) + "T";
        ret = ret + Print2Digit(d.getUTCHours()) + ":" + Print2Digit(d.getUTCMinutes()) + ":" + Print2Digit(d.getUTCSeconds()) + ".";
        let ms = d.getUTCMilliseconds();
        if(ms < 100)
        {
            if(ms < 10)
                ms = "00" + ms;
            else
                ms = "0" + ms;
        }
        ret = ret + ms + "Z";
        return ret;
    }

    function cpcsp_console_log(level, msg)
    {
        if (is_cadesplugin_defined()) {
            if (level <= cadesplugin.current_log_level) {
                if (level === cadesplugin.LOG_LEVEL_DEBUG)
                    console.log("DEBUG: %s", msg);
                if (level === cadesplugin.LOG_LEVEL_INFO)
                    console.info("INFO: %s", msg);
                if (level === cadesplugin.LOG_LEVEL_ERROR)
                    console.error("ERROR: %s", msg);
            }
        }
    }

    function set_log_level(level)
    {
        if (is_cadesplugin_defined()) {
            cadesplugin.current_log_level = level;
            if (cadesplugin.current_log_level === cadesplugin.LOG_LEVEL_DEBUG)
                cpcsp_console_log(cadesplugin.LOG_LEVEL_INFO, "log_level = DEBUG");
            if (cadesplugin.current_log_level === cadesplugin.LOG_LEVEL_INFO)
                cpcsp_console_log(cadesplugin.LOG_LEVEL_INFO, "log_level = INFO");
            if (cadesplugin.current_log_level === cadesplugin.LOG_LEVEL_ERROR)
                cpcsp_console_log(cadesplugin.LOG_LEVEL_INFO, "log_level = ERROR");
        }
    }

    function check_chrome_plugin(plugin_loaded, plugin_loaded_error)
    {
        if (is_cadesplugin_defined()) {
            cadesplugin.async_spawn(function* (args) {
                try {
                    let pluginObject = yield CreatePluginObject();
                    let oAbout = yield pluginObject.CreateObjectAsync("CAdESCOM.About");
                    cadesplugin.set(pluginObject);
                    args[0]();
                } catch (err) {
                    args[1]("Плагин недоступен");
                }
            }, plugin_loaded, plugin_loaded_error);
        }

    }
    cpcsp_chrome_nmcades.check_chrome_plugin = check_chrome_plugin;

    //значения функций для обслуживания Promise
    let g_resolve_function = {};
    let g_reject_function = {};
    let g_request_id = 1;

    function Json_to_javascript(data)
    {
        if(data.retval.type === "object")
        {
            obj = {};
            obj['objid'] = data.retval.value;
            if(typeof data.retval.properties === "object")
            {
                const props = data.retval.properties;
                for(let i = 0; i < props.length; i++)
                {
    //                Object.defineProperty(obj, props[i], {get : CallGetProperty.bind(obj, props[i]),
    //                                                      set : CallSetProperty.bind(obj, props[i])});
                    Object.defineProperty(obj, props[i], {get : CallGetProperty.bind(obj, props[i])});
                    obj["propset_" + props[i]] = CallSetProperty.bind(obj, props[i]);
                }
            }
            if(typeof data.retval.methods === "object")
            {
                const methods = data.retval.methods;
                for(let i = 0; i < methods.length; i++)
                {
                    obj[methods[i]] = CallMethod.bind(obj, methods[i]);
                }

            }
            return obj;
        }
        if(data.retval.type === "string")
        {
            return data.retval.value;
        }
        if(data.retval.type === "number")
        {
            return parseInt(data.retval.value);
        }
        if(data.retval.type === "boolean")
        {
            return Boolean(data.retval.value);
        }
        if(data.retval.type === "OK")
        {
            return;
        }
    }

    function CallMethod(){
        //create message structure
        g_request_id++;
        args = new Array();
        let arg;
        for(let i = 1; i < arguments.length; i++)
        {
            if (typeof arguments[i] === "object")
            {
                if (arguments[i] instanceof Date)
                {
                    arg = { type: "string", value: DateToUTCStr(arguments[i]) };
                    args.push(arg);
                    continue;
                }
                arg = { type: typeof arguments[i], value: arguments[i]["objid"] };
                args.push(arg);
                continue;
            }
            else if (typeof arguments[i] === "function") {
                let j = 0;
                for (; j < callbackFuncs.length; j++) {
                    if (arguments[i] === callbackFuncs[j].value) {
                        func = { type: "function", id: j, value: arguments[i] };
                        arg = { type: "number", value: func.id };
                        args.push(arg);
                        break;
                    }
                }
                if (j !== callbackFuncs.length)
                    continue;
                func = { type: "function", id: callbackFuncs.length, value: arguments[i] };
                callbackFuncs.push(func);
                arg = { type: "number", value: func.id };
                args.push(arg);
                continue;
            }
            arg = {type: typeof arguments[i], value: arguments[i]};
            args.push(arg);
        }
        object_messsage = {destination:"nmcades", requestid: g_request_id, objid: this["objid"], method: arguments[0],
                         params: args};
        cpcsp_console_log(cadesplugin.LOG_LEVEL_DEBUG, "nmcades_plugin_api.js: Sent message:" + JSON.stringify(object_messsage));
        const requestPromise = new Promise ( function (resolve, reject) {
            g_resolve_function[g_request_id] = resolve;
            g_reject_function[g_request_id] = reject;
            window.postMessage( object_messsage, "*");
        });
        return requestPromise.then(function (result) {
            return Json_to_javascript(result.data);
        });
    }

    function CreatePluginObject()
    {
        //create message structure
        g_request_id++;
        const docURL = window.document.URL;
        object_messsage = {destination:"nmcades", requestid: g_request_id, type: "init", url: docURL};
        cpcsp_console_log(cadesplugin.LOG_LEVEL_DEBUG, "nmcades_plugin_api.js: Sent message:" + JSON.stringify(object_messsage));
        const requestPromise = new Promise ( function (resolve, reject) {
            g_resolve_function[g_request_id] = resolve;
            g_reject_function[g_request_id] = reject;
            window.postMessage( object_messsage, "*");
        });
        return requestPromise.then(function (result) {
            obj = {};
            obj['objid'] = result.data.value;
            obj.CreateObjectAsync = CallMethod.bind(obj, "CreateObject");
            return obj;
        });
    }
    cpcsp_chrome_nmcades.CreatePluginObject = CreatePluginObject;

    function ReleasePluginObjects()
    {
        //create message structure
        g_request_id++;
        const docURL = window.document.URL;
        object_messsage = {destination:"nmcades", requestid: g_request_id, type: "reset", url: docURL};
        cpcsp_console_log(cadesplugin.LOG_LEVEL_DEBUG, "nmcades_plugin_api.js: Sent message:" + JSON.stringify(object_messsage));
        const requestPromise = new Promise ( function (resolve, reject) {
            g_resolve_function[g_request_id] = resolve;
            g_reject_function[g_request_id] = reject;
            window.postMessage( object_messsage, "*");
        });
        return requestPromise.then(function (result) {
            if (result.data.value == 0)
                return true;
            return false;
        });
    }
    cpcsp_chrome_nmcades.ReleasePluginObjects = ReleasePluginObjects;

    function CallGetProperty(){
        g_request_id++;
        object_messsage = {destination:"nmcades", requestid: g_request_id, objid: this['objid'], get_property: arguments[0]};
        cpcsp_console_log(cadesplugin.LOG_LEVEL_DEBUG, "nmcades_plugin_api.js: Sent message:" + JSON.stringify(object_messsage));
        const requestPromise = new Promise ( function (resolve, reject) {
            g_resolve_function[g_request_id] = resolve;
            g_reject_function[g_request_id] = reject;
            window.postMessage( object_messsage, "*");
        });
        return requestPromise.then(function (result) {
            return Json_to_javascript(result.data);
        });
    }

    function CallSetProperty(){
        g_request_id++;
        args = new Array();
        let arg;
        if(typeof arguments[1] === "object")
        {
            if(arguments[1] instanceof Date)
            {
                arg = {type: "string", value: DateToUTCStr(arguments[1])};
            }else
            {
                arg = {type: typeof arguments[1], value: arguments[1]["objid"]};
            }
        }else {
            arg = {type: typeof arguments[1], value: arguments[1]};
        }
        args.push(arg);
        object_messsage = {destination:"nmcades", requestid: g_request_id, objid: this['objid'], set_property: arguments[0], params: args};
        cpcsp_console_log(cadesplugin.LOG_LEVEL_DEBUG, "nmcades_plugin_api.js: Sent message:" + JSON.stringify(object_messsage));
        const requestPromise = new Promise ( function (resolve, reject) {
            g_resolve_function[g_request_id] = resolve;
            g_reject_function[g_request_id] = reject;
            window.postMessage( object_messsage, "*");
        });
        return requestPromise.then(function (result) {
            return Json_to_javascript(result.data);
        });
    }

    function windowListner (event){
         if (event.source !== window)
            return;
         if (event.data.tabid) {
             cpcsp_console_log(cadesplugin.LOG_LEVEL_DEBUG, "nmcades_plugin_api.js: Received message: " + JSON.stringify(event.data));
             if(event.data.data.type === "result"){
                 if(g_resolve_function[event.data.data.requestid] === null) {
                     return;
                 }
                 g_resolve_function[event.data.data.requestid](event.data);
                 g_reject_function[event.data.data.requestid] = null;
                 g_resolve_function[event.data.data.requestid] = null;
             }
             else if(event.data.data.type === "error"){
                 g_reject_function[event.data.data.requestid](event.data.data);
                 g_resolve_function[event.data.data.requestid] = null;
                 g_reject_function[event.data.data.requestid] = null;
             }
         }
    }

    function EnableInternalCSPListener(event){
        if (event.source !== window)
            return;
        if (event.data === "EnableInternalCSP_request") {
            if (is_cadesplugin_defined())
                window.postMessage("EnableInternalCSP=" + cadesplugin.EnableInternalCSP, "*");
        }
    }

    function storeCallback(event){
        if (event.source !== window)
            return;
        if (event.data.object !== undefined) {
            var certobj = JSON.parse(event.data.object);
            var cert = Json_to_javascript(certobj.data);
            var id = event.data.value;
            callbackFuncs[id].value(cert);
            return;
        }
    }

    window.addEventListener("message", windowListner, false);
    window.addEventListener("message", EnableInternalCSPListener, false);
    window.addEventListener("message", storeCallback, false);
    if (is_cadesplugin_defined())
        window.postMessage("EnableInternalCSP=" + cadesplugin.EnableInternalCSP, "*");
    window.cpcsp_chrome_nmcades = cpcsp_chrome_nmcades;
}());
