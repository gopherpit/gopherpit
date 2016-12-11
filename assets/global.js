"use strict";

axios.defaults.withCredentials = true;
axios.defaults.xsrfCookieName = "secid";
axios.defaults.xsrfHeaderName = "X-SecID";

function standardHTTPError(store, response) {
    var func = function(response) {
        if (response.data === null) {
            store.errors = ["Error communicating with the server."];
            return
        }
        switch (response.status) {
            case 401:
                store.errors = ["Invalid credentials."];
                break;
            case 403:
                store.errors = ["Forbidden."];
                break;
            default:
              if (response.data.errors === undefined) {
                  store.errors = ["Server error."];
              } else {
                  store.errors = response.data.errors || [];
              }
        }
        store.fieldErrors = response.data["field-errors"] || {};
    }
    if (response != undefined) {
        return func(response)
    }
    return func
};

function httpRequest(store, method, url, success, fail) {
    store.isLoading = true;
    store.errors = [];
    store.fieldErrors = {};
    if (fail === undefined) {
        fail = standardHTTPError(store)
    }
    axios({
        method: method,
        url: url,
        data: store.fields
    }).then(success).catch(function (error) {
        if (error.response) {
            fail(error.response)
        } else {
            store.errors = [error.message]
        }
    }).then(function(){
        store.isLoading = false;
    })
}

function httpPost(vue, url, success, fail) {
    httpRequest(vue, "POST", url, success, fail)
}

function httpDelete(vue, url, success, fail) {
    httpRequest(vue, "DELETE", url, success, fail)
}
