package jsonresponse

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type MessageResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func Respond(w http.ResponseWriter, statusCode int, response interface{}) {
	if response == nil {
		response = &MessageResponse{}
	} else if r, ok := response.(MessageResponse); ok {
		response = &r
	}
	if r, ok := response.(*MessageResponse); ok {
		if r.Code == 0 {
			r.Code = statusCode
		}
		if r.Message == "" {
			r.Message = http.StatusText(statusCode)
		}
	}
	b, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	if DefaultContentTypeHeader != "" {
		w.Header().Set("Content-Type", DefaultContentTypeHeader)
	}
	w.WriteHeader(statusCode)
	fmt.Fprint(w, string(b)+"\n")
}

// 1xx

func Continue(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusContinue, response)
}

func SwitchingProtocols(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusSwitchingProtocols, response)
}

// 2xx

func OK(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusOK, response)
}

func Created(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusCreated, response)
}

func Accepted(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusAccepted, response)
}

func NonAuthoritativeInfo(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusNonAuthoritativeInfo, response)
}

func NoContent(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusNoContent, response)
}

func ResetContent(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusResetContent, response)
}

func PartialContent(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusPartialContent, response)
}

// 3xx

func MultipleChoices(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusMultipleChoices, response)
}

func MovedPermanently(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusMovedPermanently, response)
}

func Found(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusFound, response)
}

func SeeOther(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusSeeOther, response)
}

func NotModified(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusNotModified, response)
}

func UseProxy(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusUseProxy, response)
}

func TemporaryRedirect(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusTemporaryRedirect, response)
}

// 4xx

func BadRequest(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusBadRequest, response)
}

func Unauthorized(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusUnauthorized, response)
}

func PaymentRequired(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusPaymentRequired, response)
}

func Forbidden(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusForbidden, response)
}

func NotFound(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusNotFound, response)
}

func MethodNotAllowed(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusMethodNotAllowed, response)
}

func NotAcceptable(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusNotAcceptable, response)
}

func ProxyAuthRequired(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusProxyAuthRequired, response)
}

func RequestTimeout(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusRequestTimeout, response)
}

func Conflict(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusConflict, response)
}

func Gone(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusGone, response)
}

func LengthRequired(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusLengthRequired, response)
}

func PreconditionFailed(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusPreconditionFailed, response)
}

func RequestEntityTooLarge(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusRequestEntityTooLarge, response)
}

func RequestURITooLong(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusRequestURITooLong, response)
}

func UnsupportedMediaType(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusUnsupportedMediaType, response)
}

func RequestedRangeNotSatisfiable(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusRequestedRangeNotSatisfiable, response)
}

func ExpectationFailed(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusExpectationFailed, response)
}

func Teapot(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusTeapot, response)
}

// 5xx

func InternalServerError(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusInternalServerError, response)
}

func NotImplemented(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusNotImplemented, response)
}

func BadGateway(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusBadGateway, response)
}

func ServiceUnavailable(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusServiceUnavailable, response)
}

func GatewayTimeout(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusGatewayTimeout, response)
}

func HTTPVersionNotSupported(w http.ResponseWriter, response interface{}) {
	Respond(w, http.StatusHTTPVersionNotSupported, response)
}
