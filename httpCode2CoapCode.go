package main

import (
	coap "github.com/go-ocf/go-coap"
	"github.com/valyala/fasthttp"
)

func httpCode2CoapCode(statusCode int, method coap.COAPCode) coap.COAPCode {
	switch statusCode {
	//1xx
	case fasthttp.StatusContinue:
		return coap.Continue
	case fasthttp.StatusSwitchingProtocols:
	case fasthttp.StatusProcessing:

	//2xx
	case fasthttp.StatusOK:
		switch method {
		case coap.POST:
			return coap.Changed
		case coap.GET:
			return coap.Content
		case coap.PUT:
			return coap.Created
		case coap.DELETE:
			return coap.Deleted
		}
	case fasthttp.StatusCreated:
		return coap.Created
	case fasthttp.StatusAccepted:
	case fasthttp.StatusNonAuthoritativeInfo:
	case fasthttp.StatusNoContent:
	case fasthttp.StatusResetContent:
	case fasthttp.StatusPartialContent:
	case fasthttp.StatusMultiStatus:
	case fasthttp.StatusAlreadyReported:
	case fasthttp.StatusIMUsed:

	//3xx
	case fasthttp.StatusMultipleChoices:
	case fasthttp.StatusMovedPermanently:
	case fasthttp.StatusFound:
	case fasthttp.StatusSeeOther:
	case fasthttp.StatusNotModified:
	case fasthttp.StatusUseProxy:
	case fasthttp.StatusTemporaryRedirect:
	case fasthttp.StatusPermanentRedirect:

	//4xx
	case fasthttp.StatusBadRequest:
		return coap.BadRequest
	case fasthttp.StatusUnauthorized:
		return coap.Unauthorized
	case fasthttp.StatusPaymentRequired:
	case fasthttp.StatusForbidden:
		return coap.Forbidden
	case fasthttp.StatusNotFound:
		return coap.NotFound
	case fasthttp.StatusMethodNotAllowed:
		return coap.MethodNotAllowed
	case fasthttp.StatusNotAcceptable:
		return coap.NotAcceptable
	case fasthttp.StatusProxyAuthRequired:
	case fasthttp.StatusRequestTimeout:
	case fasthttp.StatusConflict:
	case fasthttp.StatusGone:
	case fasthttp.StatusLengthRequired:
	case fasthttp.StatusPreconditionFailed:
		return coap.PreconditionFailed
	case fasthttp.StatusRequestEntityTooLarge:
		return coap.RequestEntityTooLarge
	case fasthttp.StatusRequestURITooLong:
	case fasthttp.StatusUnsupportedMediaType:
		return coap.UnsupportedMediaType
	case fasthttp.StatusRequestedRangeNotSatisfiable:
	case fasthttp.StatusExpectationFailed:
	case fasthttp.StatusTeapot:
	case fasthttp.StatusUnprocessableEntity:
	case fasthttp.StatusLocked:
	case fasthttp.StatusFailedDependency:
	case fasthttp.StatusUpgradeRequired:
	case fasthttp.StatusPreconditionRequired:
	case fasthttp.StatusTooManyRequests:
	case fasthttp.StatusRequestHeaderFieldsTooLarge:
	case fasthttp.StatusUnavailableForLegalReasons:

	//5xx
	case fasthttp.StatusInternalServerError:
	case fasthttp.StatusNotImplemented:
		return coap.NotImplemented
	case fasthttp.StatusBadGateway:
		return coap.BadGateway
	case fasthttp.StatusServiceUnavailable:
		return coap.ServiceUnavailable
	case fasthttp.StatusGatewayTimeout:
		return coap.GatewayTimeout
	case fasthttp.StatusHTTPVersionNotSupported:
	case fasthttp.StatusVariantAlsoNegotiates:
	case fasthttp.StatusInsufficientStorage:
	case fasthttp.StatusLoopDetected:
	case fasthttp.StatusNotExtended:
	case fasthttp.StatusNetworkAuthenticationRequired:
	}
	return coap.InternalServerError
}
