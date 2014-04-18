// from index.html, for exposing internals for testing
var shform = shform || {};

$(document).ready(function () {
    "use strict";

    var wsconn,                     // WebSocket connection
    last_received,
    send = function (msg) {
        if (msg === last_received) {
            console.log("not sending last received");
            return;
        }
        if (wsconn) {
            console.log("sending message (" + msg + ") to ws");
            wsconn.send(msg);
        } else {
            console.log("no ws.  NOT sending message (" + msg + ")");
        }
    };

    if (window["WebSocket"]) {
        console.log("setting up WebSocket");
        wsconn = new WebSocket("{{$}}");
        wsconn.onclose = function(evt) {
            console.log("connection closed");
        }
        wsconn.onmessage = function(evt) {
            console.log("received: " + evt.data);
            last_received = evt.data;
            shform.viewModel.bandVal(evt.data);
        }
        shform.wsconn = wsconn;
    } else {
        // XXXtodo: Add fall-back (e.g., to long polling) here.
        $("p.lead").html("Sorry.  Your browser does not support WebSockets.");
        console.log("no ws support in browser");
    }

    // based on example in Knockout docs:
    // http://knockoutjs.com/documentation/rateLimit-observable.html
    function AppViewModel() {
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
            send(val);
        }, this);
    }

    shform.viewModel = new AppViewModel();
    ko.applyBindings(shform.viewModel, $('#bandDiv').get(0));
});
