// from index.html, for exposing internals for testing
var shform = shform || {};

$(document).ready(function () {
    "use strict";

    var wsconn,                 // WebSocket connection
    prop,                       // property
    lastReceived,
    send = function (vm, prop, val) {
        var viewModel = shform.viewModels[vm],
        msg = JSON.stringify({
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
                console.log("connection closed");
            }
            wsconn.onmessage = function(evt) {
                console.log("received: " + evt.data);
                msg = $.parseJSON(evt.data);
                lastReceived = JSON.stringify(msg);
                viewModel = shform.viewModels[msg.vm];
                viewModel[msg.prop](msg.val);
            }
            shform.wsconn = wsconn;
        } else {
            // XXXtodo: Add fall-back (e.g., to long polling) here.
            $("p.lead").html("Sorry.  Your browser does not support WebSockets.");
            console.log("no ws support in browser");
        }
    };

    setupWs();

    // based on example in Knockout docs:
    // http://knockoutjs.com/documentation/rateLimit-observable.html
    function BandViewModel() {
        this.name = "band";
        this.bandVal = ko.observable();
        this.bandSlowVal = ko.computed(this.bandVal)
            .extend({
                rateLimit: {
                    method: "notifyWhenChangesStop",
                    timeout: 400
                }
            });

        // Keep a log of the throttled values passed to the WebSocket.
        this.loggedValues = ko.observableArray([]);
        this.bandSlowVal.subscribe(function (val) {
            this.loggedValues.push(val);
            send(this.name, "bandVal", val);
        }, this);
    }

    shform.viewModels = {};
    shform.viewModels.band = new BandViewModel();
    for (prop in shform.viewModels) {
        if (!shform.viewModels.hasOwnProperty(prop)) {
            continue;
        }
        ko.applyBindings(shform.viewModels[prop], $('#' + prop + 'Div').get(0));
    }
});
