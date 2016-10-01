package jsonresponse // import "resenje.org/jsonresponse"

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	DefaultMessageKey           = "message"
	DefaultCodeKey              = "code"
	DefaultProgrammingExcuseKey = "programming-excuse"
	DefaultContentTypeHeader    = "application/json; charset=utf-8"
)

type Response map[string]interface{}

func (r Response) String() (s string) {
	b, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (r Response) Response(w http.ResponseWriter, statusCode int) {
	if _, ok := r[DefaultMessageKey]; !ok {
		r[DefaultMessageKey] = http.StatusText(statusCode)
	}
	if _, ok := r[DefaultCodeKey]; !ok {
		r[DefaultCodeKey] = statusCode
	}
	if DefaultContentTypeHeader != "" {
		w.Header().Set("Content-Type", DefaultContentTypeHeader)
	}
	w.WriteHeader(statusCode)
	fmt.Fprint(w, r.String()+"\n")
}

func (r Response) WithProgrammingExcuse() Response {
	r[DefaultProgrammingExcuseKey] = randomExcuse()
	return r
}

// 1xx

func (r Response) Continue(w http.ResponseWriter) {
	r.Response(w, http.StatusContinue)
}

func (r Response) SwitchingProtocols(w http.ResponseWriter) {
	r.Response(w, http.StatusSwitchingProtocols)
}

// 2xx

func (r Response) OK(w http.ResponseWriter) {
	r.Response(w, http.StatusOK)
}

func (r Response) Created(w http.ResponseWriter) {
	r.Response(w, http.StatusCreated)
}

func (r Response) Accepted(w http.ResponseWriter) {
	r.Response(w, http.StatusAccepted)
}

func (r Response) NonAuthoritativeInfo(w http.ResponseWriter) {
	r.Response(w, http.StatusNonAuthoritativeInfo)
}

func (r Response) NoContent(w http.ResponseWriter) {
	r.Response(w, http.StatusNoContent)
}

func (r Response) ResetContent(w http.ResponseWriter) {
	r.Response(w, http.StatusResetContent)
}

func (r Response) PartialContent(w http.ResponseWriter) {
	r.Response(w, http.StatusPartialContent)
}

// 3xx

func (r Response) MultipleChoices(w http.ResponseWriter) {
	r.Response(w, http.StatusMultipleChoices)
}

func (r Response) MovedPermanently(w http.ResponseWriter) {
	r.Response(w, http.StatusMovedPermanently)
}

func (r Response) Found(w http.ResponseWriter) {
	r.Response(w, http.StatusFound)
}

func (r Response) SeeOther(w http.ResponseWriter) {
	r.Response(w, http.StatusSeeOther)
}

func (r Response) NotModified(w http.ResponseWriter) {
	r.Response(w, http.StatusNotModified)
}

func (r Response) UseProxy(w http.ResponseWriter) {
	r.Response(w, http.StatusUseProxy)
}

func (r Response) TemporaryRedirect(w http.ResponseWriter) {
	r.Response(w, http.StatusTemporaryRedirect)
}

// 4xx

func (r Response) BadRequest(w http.ResponseWriter) {
	r.Response(w, http.StatusBadRequest)
}

func (r Response) Unauthorized(w http.ResponseWriter) {
	r.Response(w, http.StatusUnauthorized)
}

func (r Response) PaymentRequired(w http.ResponseWriter) {
	r.Response(w, http.StatusPaymentRequired)
}

func (r Response) Forbidden(w http.ResponseWriter) {
	r.Response(w, http.StatusForbidden)
}

func (r Response) NotFound(w http.ResponseWriter) {
	r.Response(w, http.StatusNotFound)
}

func (r Response) MethodNotAllowed(w http.ResponseWriter) {
	r.Response(w, http.StatusMethodNotAllowed)
}

func (r Response) NotAcceptable(w http.ResponseWriter) {
	r.Response(w, http.StatusNotAcceptable)
}

func (r Response) ProxyAuthRequired(w http.ResponseWriter) {
	r.Response(w, http.StatusProxyAuthRequired)
}

func (r Response) RequestTimeout(w http.ResponseWriter) {
	r.Response(w, http.StatusRequestTimeout)
}

func (r Response) Conflict(w http.ResponseWriter) {
	r.Response(w, http.StatusConflict)
}

func (r Response) Gone(w http.ResponseWriter) {
	r.Response(w, http.StatusGone)
}

func (r Response) LengthRequired(w http.ResponseWriter) {
	r.Response(w, http.StatusLengthRequired)
}

func (r Response) PreconditionFailed(w http.ResponseWriter) {
	r.Response(w, http.StatusPreconditionFailed)
}

func (r Response) RequestEntityTooLarge(w http.ResponseWriter) {
	r.Response(w, http.StatusRequestEntityTooLarge)
}

func (r Response) RequestURITooLong(w http.ResponseWriter) {
	r.Response(w, http.StatusRequestURITooLong)
}

func (r Response) UnsupportedMediaType(w http.ResponseWriter) {
	r.Response(w, http.StatusUnsupportedMediaType)
}

func (r Response) RequestedRangeNotSatisfiable(w http.ResponseWriter) {
	r.Response(w, http.StatusRequestedRangeNotSatisfiable)
}

func (r Response) ExpectationFailed(w http.ResponseWriter) {
	r.Response(w, http.StatusExpectationFailed)
}

func (r Response) Teapot(w http.ResponseWriter) {
	r.Response(w, http.StatusTeapot)
}

// 5xx

func (r Response) InternalServerError(w http.ResponseWriter) {
	r.Response(w, http.StatusInternalServerError)
}

func (r Response) NotImplemented(w http.ResponseWriter) {
	r.Response(w, http.StatusNotImplemented)
}

func (r Response) BadGateway(w http.ResponseWriter) {
	r.Response(w, http.StatusBadGateway)
}

func (r Response) ServiceUnavailable(w http.ResponseWriter) {
	r.Response(w, http.StatusServiceUnavailable)
}

func (r Response) GatewayTimeout(w http.ResponseWriter) {
	r.Response(w, http.StatusGatewayTimeout)
}

func (r Response) HTTPVersionNotSupported(w http.ResponseWriter) {
	r.Response(w, http.StatusHTTPVersionNotSupported)
}
