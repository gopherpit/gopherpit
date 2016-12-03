"use strict";

Vue.http.options.xhr = {withCredentials: true};

Vue.http.interceptors.push(function(request, next) {
    if (!xsrfSafeMethod(request.method) && sameOrigin(request.url)) {
        var cookie = getCookie("secid");
        if (cookie != "") request.headers.set("X-SecID", cookie);
    };
    next();
});

function getCookie(name) {
    var cookieValue = "";
    if (document.cookie && document.cookie != '') {
        var cookies = document.cookie.split(';');
        for (var i = 0; i < cookies.length; i++) {
            var cookie = cookies[i].replace(/^\s+|\s+$/g, '');;
            if (cookie.substring(0, name.length + 1) == (name + '=')) {
                cookieValue = decodeURIComponent(cookie.substring(name.length + 1));
                break;
            }
        }
    }
    return cookieValue;
};

function xsrfSafeMethod(method) {
    return (/^(GET|HEAD|OPTIONS|TRACE)$/.test(method));
};

function sameOrigin(url) {
    var host = document.location.host;
    var protocol = document.location.protocol;
    var sr_origin = '//' + host;
    var origin = protocol + sr_origin;
    return (url == origin || url.slice(0, origin.length + 1) == origin + '/') ||
        (url == sr_origin || url.slice(0, sr_origin.length + 1) == sr_origin + '/') ||
        !(/^(\/\/|http:|https:).*/.test(url));
};

function standardHTTPError(response) {
    if (response.data === null) {
        this.errors = ["Error communicating with the server."];
        return
    }
    switch (response.status) {
        case 401:
            this.errors = ["Invalid credentials."];
            break;
        case 403:
            this.errors = ["Forbidden."];
            break;
        default:
          if (response.data.errors === undefined) {
              this.errors = ["Server error."];
          } else {
              this.errors = response.data.errors || [];
          }
    }
    this.fieldErrors = response.data["field-errors"] || {};
};

function httpRequest(vue, method, url, success) {
    vue.isLoading = true;
    vue.errors = [];
    vue.fieldErrors = {};
    if (method == "POST") {
        vue.$http.post(url, vue.fields).then(success, standardHTTPError).then(function(){
            vue.isLoading = false;
        })
    } else if (method == "DELETE") {
        vue.$http.delete(url, vue.fields).then(success, standardHTTPError).then(function(){
            vue.isLoading = false;
        })
    }
}

function httpPost(vue, url, success) {
    httpRequest(vue, "POST", url, success)
}

function httpDelete(vue, url, success) {
    httpRequest(vue, "DELETE", url, success)
}
