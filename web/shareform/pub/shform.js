$(document).ready(function () {

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

        // Keep a log of the throttled values
        this.loggedValues = ko.observableArray([]);
        this.delayedValue.subscribe(function (val) {
            if (val !== '')
                this.loggedValues.push(val);
        }, this);
    }

    ko.applyBindings(new AppViewModel());
});
