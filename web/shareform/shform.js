// from index.html, for exposing internals for testing
var shform = shform || {};

$(document).ready(function () {
    "use strict";

    var wsconn,                 // WebSocket connection
    send = function (msg) {
        if (wsconn) {
            console.log("sending message (" + msg + ") to ws");
            wsconn.send(msg);
        } else {
            console.log("NOT sending message (" + msg + ") to ws");
        }
    };

    if (window["WebSocket"]) {
        // XXXtodo: Use URL from templating.
        console.log("setting up WebSocket");
        wsconn = new WebSocket("ws://127.0.0.1:8181/ws");
        wsconn.onclose = function(evt) {
            console.log("connection closed");
        }
        wsconn.onmessage = function(evt) {
            console.log("received: " + evt.data);
        }
        shform.wsconn = wsconn;
    } else {
        // XXX fall-back (e.g., to long polling) here.
        console.log("no ws support in browser");
    }

    // based on example in Knockout docs:
    // http://knockoutjs.com/documentation/rateLimit-observable.html
    function AppViewModel() {
        this.instantaneousValue = ko.observable();
        this.delayedValue = ko.computed(this.instantaneousValue)
            .extend({
                rateLimit: {
                    method: "notifyWhenChangesStop",
                    timeout: 400
                }
            });

        // Keep a log of the throttled values passed to the WebSocket.
        this.loggedValues = ko.observableArray([]);
        this.delayedValue.subscribe(function (val) {
            if (val !== '') {
                this.loggedValues.push(val);
                send(val);
            }
        }, this);
    }

    ko.applyBindings(new AppViewModel());
});
