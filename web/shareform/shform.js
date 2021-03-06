// from index.html, for exposing internals for testing
var shform = shform || {};

$(document).ready(function () {
    "use strict";

    var wsconn,                 // WebSocket connection
    prop,                       // property
    lastReceived,
    send = function (vm, prop, val) {
        var msg = JSON.stringify({
            "vm" : vm,
            "prop" : prop,
            "val" : val
        });

        if (msg === lastReceived) {
            console.log("not sending last received");
            return;
        }
        if (wsconn) {
            console.log("sending message (" + msg + ") to ws");
            wsconn.send(msg);
        } else {
            console.log("no ws.  NOT sending message (" + msg + ")");
        }
    },
    setupWs = function () {
        var msg, viewModel;

        if (window["WebSocket"]) {
            console.log("setting up WebSocket");
            wsconn = new WebSocket("{{$}}");
            wsconn.onclose = function(evt) {
                console.log("connection closed: " + evt);
            };
            wsconn.onmessage = function(evt) {
                console.log("received: " + evt.data);
                msg = $.parseJSON(evt.data);
                lastReceived = JSON.stringify(msg);
                viewModel = shform.viewModels[msg.vm];
                if (viewModel.doHighlight) {
                    viewModel.doHighlight();
                }
                viewModel[msg.prop](msg.val);
            };
            shform.wsconn = wsconn;
        } else {
            // XXXtodo: Add fall-back (e.g., to long polling) here.
            $("p.lead").html("Sorry.  Your browser does not support WebSockets.");
            console.log("no ws support in browser");
        }
    },
    highlightFn = function (vm) {
        return function () {
            var sel = "#" + vm.name + "Div",
            fadeStep = 100,
            fade = function () {
                setTimeout(function () {
                    vm.hlalpha = vm.hlalpha * 0.45;
                    if (vm.hlalpha < 0.001) {
                        vm.hlalpha = 0;
                    } else {
                        setTimeout(fade, fadeStep);
                    }
                    $(sel)
                        .css("border",
                             "3px solid rgba(228, 255, 77, "+vm.hlalpha+")");
                }, fadeStep);
            };

            vm.hlalpha = vm.hlmax;
            $(sel)
                .css("border",
                     "3px solid rgba(228, 255, 77, "+vm.hlalpha+")");
            setTimeout(fade, fadeStep);
        };
    },
    vmSetDefaults = function (vm) {
        vm.hlmax = 150;       // maximum highlighting alpha
        vm.hlalpha = 0;       // highlighting alpha
        vm.doHighlight = highlightFn(vm);
        return vm;
    };

    setupWs();

    // based on example in Knockout docs:
    // http://knockoutjs.com/documentation/rateLimit-observable.html
    function BandViewModel() {
        var self = this;

        vmSetDefaults(self);
        self.name = "band";
        self.bandVal = ko.observable();
        self.bandSlowVal = ko.computed(self.bandVal)
            .extend({
                rateLimit: {
                    method: "notifyWhenChangesStop",
                    timeout: 400
                }
            });
        self.bandSlowVal.subscribe(function (val) {
            send(self.name, "bandVal", val);
        }, self);
    };

    shform.viewModels = {};
    shform.viewModels.instrument = vmSetDefaults({
        "name" : "instrument",
        "instSel" : ko.observable("Guitar")
    });
    shform.viewModels.instrument.instSel.subscribe(function (sel) {
        send('instrument', 'instSel', sel);
    });
    shform.viewModels.electric = vmSetDefaults({
        "name" : "electric",
        "electricSel" : ko.observable("electric")
    });
    shform.viewModels.electric.electricSel.subscribe(function (sel) {
        send('electric', 'electricSel', sel);
    });
    shform.viewModels.feeling = vmSetDefaults({
        "name" : "feeling",
        "withFeeling" : ko.observable(true)
    });
    shform.viewModels.feeling.withFeeling.subscribe(function (yesno) {
        send('feeling', 'withFeeling', yesno);
    });
    shform.viewModels.band = new BandViewModel();
    for (prop in shform.viewModels) {
        if (shform.viewModels.hasOwnProperty(prop)) {
            ko.applyBindings(shform.viewModels[prop],
                             $('#' + prop + 'Div').get(0));
        }
    }
});
