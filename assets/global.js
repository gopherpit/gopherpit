"use strict";

$(function(){
    $.ajaxSetup({
        beforeSend: function(xhr, settings) {
            if (!xsrfSafeMethod(settings.type) && sameOrigin(settings.url)) {
                xhr.setRequestHeader("X-SecID", getCookie("secid"));
            };
            xhr.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
            xhr.withCredentials = true;
        }
    });
    $('.nav-toggle').click(function() {
      $(this).toggleClass('is-active');
      $(this).siblings('.nav-menu').toggleClass('is-active');
    });
    $('body').on('click', '.notification .delete', function(e){
      $(this).closest('.notification').fadeOut('fast')
    });
    $('body').on('click', '.modal-button', function() {
       var target = $(this).data('target');
       $('html').addClass('is-clipped');
       $(target).addClass('is-active');
    });
    $('body').on('click', '.modal-background, .modal-close, .modal-close-button', function() {
      $('html').removeClass('is-clipped');
      $(this).closest('.modal').removeClass('is-active');
    });
    $.fn.ajaxform = function(options) {
        if (options === undefined) {
            var options = {};
        }
        var $form = this;
        return $form.submit(function(e) {
            e.preventDefault();
            var data = formToObject($form);
            var fn = options["before"];
            if (fn !== undefined) fn(data);
            data = JSON.stringify(data);
            $form.addClass("is-disabled");
            $form.find("input, button, select").prop("disabled", true);
            $form.find("label").addClass("is-disabled");
            $form.find("[type=submit]").addClass("is-loading");
            $form.find(".help.is-danger, .notification").remove();
            $form.find(".is-danger").removeClass("is-danger");
            $.ajax({
                type: $form.attr('method') || 'post',
                url: $form.attr('action'),
                data: data,
                dataType: 'json'
            }).done(function(data, textStatus, jqXHR) {
                var fn = options["ok"];
                if (fn !== undefined) fn(data, textStatus, jqXHR);
            }).fail(function(jqXHR, textStatus, errorThrown) {
                var fn = options['err'+jqXHR.status];
                if (fn !== undefined) {
                    fn(jqXHR, textStatus, errorThrown);
                    return
                }
                switch (jqXHR.status) {
                    case 400:
                        var errors = jqXHR.responseJSON['errors'];
                        if (errors !== undefined) {
                            var s = '';
                            $.each(errors, function(){
                                s += newNotification(this, 'is-danger')[0].outerHTML;
                            });
                            $form.prepend(s);
                        };
                        var fieldErrors = jqXHR.responseJSON['field-errors'];
                        if (fieldErrors !== undefined) {
                            $.each(fieldErrors, function(key, errors){
                                var $field = $('[name="'+key+'"]');
                                $field.addClass('is-danger');
                                $.each(errors, function(){
                                    $field.closest('.control').after($('<p>', {class: 'control help is-danger'}).text(this));
                                });
                            });
                        };
                        var fn = options['err400after'];
                        if (fn !== undefined) {
                            fn(jqXHR, textStatus, errorThrown);
                        }
                        break;
                    default:
                      var e = (errorThrown != "") ? errorThrown : "Communication error";
                      $form.prepend(newNotification(e+".", 'is-danger'))
                }
            }).always(function(){
                var fn = options["always"];
                if (fn !== undefined) fn(data, textStatus, jqXHR);
                $form.removeClass("is-disabled");
                $form.find("input, button, select").prop("disabled", false);
                $form.find("label").removeClass("is-disabled");
                $form.find("[type=submit]").removeClass("is-loading");
            });
        });
    };
    $.fn.ajaxbutton = function(options) {
        if (options === undefined) {
            var options = {};
        }
        var $obj = this;
        return $obj.click(function(e) {
            e.preventDefault();
            var data = {};
            var fn = options["before"];
            if (fn !== undefined) fn(data);
            $obj.addClass("is-disabled is-loading");
            $obj.siblings('.notification').remove();
            $.ajax({
                type: $obj.attr('data-method') || 'post',
                url: $obj.attr('data-action'),
                data: data,
                dataType: 'json'
            }).done(function(data, textStatus, jqXHR) {
                var fn = options["ok"];
                if (fn !== undefined) fn(data, textStatus, jqXHR);
            }).fail(function(jqXHR, textStatus, errorThrown) {
                var fn = options['err'+jqXHR.status];
                if (fn !== undefined) {
                    fn(jqXHR, textStatus, errorThrown);
                    return
                }
                switch (jqXHR.status) {
                    case 400:
                        var errors = jqXHR.responseJSON['errors'];
                        if (errors !== undefined) {
                            var s = '';
                            $.each(errors, function(){
                                s += newNotification(this, 'is-danger')[0].outerHTML;
                            });
                            $obj.before(s);
                        };
                        break;
                    default:
                      var e = (errorThrown != "") ? errorThrown : "Communication error";
                      $obj.before(newNotification(e+".", 'is-danger'))
                }
            }).always(function(){
                var fn = options["always"];
                if (fn !== undefined) fn(data, textStatus, jqXHR);
                $obj.removeClass("is-disabled is-loading");
            });
        });
    };
});

function newNotification(text, cls) {
    if (cls === undefined) {
        cls = "is-success"
    }
    return $('<div>', {class: 'notification '+cls}).text(text).prepend($('<div>', {class: 'delete'}))
}

function getCookie(name) {
    var cookieValue = null;
    if (document.cookie && document.cookie != '') {
        var cookies = document.cookie.split(';');
        for (var i = 0; i < cookies.length; i++) {
            var cookie = jQuery.trim(cookies[i]);
            // Does this cookie string begin with the name we want?
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
    // test that a given url is a same-origin URL
    // url could be relative or scheme relative or absolute
    var host = document.location.host; // host + port
    var protocol = document.location.protocol;
    var sr_origin = '//' + host;
    var origin = protocol + sr_origin;
    // Allow absolute or scheme relative URLs to same origin
    return (url == origin || url.slice(0, origin.length + 1) == origin + '/') ||
        (url == sr_origin || url.slice(0, sr_origin.length + 1) == sr_origin + '/') ||
        // or any other URL that isn't scheme relative or absolute i.e relative.
        !(/^(\/\/|http:|https:).*/.test(url));
};

function formToObject(form){
    var array = $(form).serializeArray();
    var obj = {};
    $.each(array, function() {
        obj[this.name] = this.value || '';
    });
    return obj;
};