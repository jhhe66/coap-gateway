package service

import (
	"testing"

	coap "github.com/go-ocf/go-coap"
	"github.com/valyala/fasthttp"
)

func TestHttpCode2CoapCode(t *testing.T) {
	tbl := []struct {
		name         string
		inStatusCode int
		inCoapCode   coap.COAPCode
		out          coap.COAPCode
	}{
		{"Continue", fasthttp.StatusContinue, coap.Empty, coap.Continue},
		{"fasthttp.StatusSwitchingProtocols", fasthttp.StatusSwitchingProtocols, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusProcessing", fasthttp.StatusProcessing, coap.Empty, coap.InternalServerError},

		//2xx
		{"fasthttp.StatusOK", fasthttp.StatusOK, coap.POST, coap.Changed},
		{"fasthttp.StatusOK", fasthttp.StatusOK, coap.GET, coap.Content},
		{"fasthttp.StatusOK", fasthttp.StatusOK, coap.PUT, coap.Created},
		{"fasthttp.StatusOK", fasthttp.StatusOK, coap.DELETE, coap.Deleted},

		{"fasthttp.StatusCreated", fasthttp.StatusCreated, coap.Empty, coap.Created},
		{"fasthttp.StatusAccepted", fasthttp.StatusAccepted, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusNonAuthoritativeInfo", fasthttp.StatusNonAuthoritativeInfo, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusNoContent", fasthttp.StatusNoContent, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusResetContent", fasthttp.StatusResetContent, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusPartialContent", fasthttp.StatusPartialContent, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusMultiStatus", fasthttp.StatusMultiStatus, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusAlreadyReported", fasthttp.StatusAlreadyReported, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusIMUsed", fasthttp.StatusIMUsed, coap.Empty, coap.InternalServerError},

		//3xx
		{"fasthttp.StatusMultipleChoices", fasthttp.StatusMultipleChoices, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusMovedPermanently", fasthttp.StatusMovedPermanently, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusFound", fasthttp.StatusFound, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusSeeOther", fasthttp.StatusSeeOther, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusNotModified", fasthttp.StatusNotModified, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusUseProxy", fasthttp.StatusUseProxy, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusTemporaryRedirect", fasthttp.StatusTemporaryRedirect, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusPermanentRedirect", fasthttp.StatusPermanentRedirect, coap.Empty, coap.InternalServerError},

		//4xx
		{"fasthttp.StatusBadRequest", fasthttp.StatusBadRequest, coap.Empty, coap.BadRequest},
		{"fasthttp.StatusUnauthorized", fasthttp.StatusUnauthorized, coap.Empty, coap.Unauthorized},
		{"fasthttp.StatusPaymentRequired", fasthttp.StatusPaymentRequired, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusForbidden", fasthttp.StatusForbidden, coap.Empty, coap.Forbidden},
		{"fasthttp.StatusNotFound", fasthttp.StatusNotFound, coap.Empty, coap.NotFound},
		{"fasthttp.StatusMethodNotAllowed", fasthttp.StatusMethodNotAllowed, coap.Empty, coap.MethodNotAllowed},
		{"fasthttp.StatusNotAcceptable", fasthttp.StatusNotAcceptable, coap.Empty, coap.NotAcceptable},
		{"fasthttp.StatusProxyAuthRequired", fasthttp.StatusProxyAuthRequired, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusRequestTimeout", fasthttp.StatusRequestTimeout, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusConflict", fasthttp.StatusConflict, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusGone", fasthttp.StatusGone, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusLengthRequired", fasthttp.StatusLengthRequired, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusPreconditionFailed", fasthttp.StatusPreconditionFailed, coap.Empty, coap.PreconditionFailed},
		{"fasthttp.StatusRequestEntityTooLarge", fasthttp.StatusRequestEntityTooLarge, coap.Empty, coap.RequestEntityTooLarge},
		{"fasthttp.StatusRequestURITooLong", fasthttp.StatusRequestURITooLong, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusUnsupportedMediaType", fasthttp.StatusUnsupportedMediaType, coap.Empty, coap.UnsupportedMediaType},
		{"fasthttp.StatusRequestedRangeNotSatisfiable", fasthttp.StatusRequestedRangeNotSatisfiable, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusExpectationFailed", fasthttp.StatusExpectationFailed, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusTeapot", fasthttp.StatusTeapot, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusUnprocessableEntity", fasthttp.StatusUnprocessableEntity, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusLocked", fasthttp.StatusLocked, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusFailedDependency", fasthttp.StatusFailedDependency, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusUpgradeRequired", fasthttp.StatusUpgradeRequired, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusPreconditionRequired", fasthttp.StatusPreconditionRequired, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusTooManyRequests", fasthttp.StatusTooManyRequests, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusRequestHeaderFieldsTooLarge", fasthttp.StatusRequestHeaderFieldsTooLarge, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusUnavailableForLegalReasons", fasthttp.StatusUnavailableForLegalReasons, coap.Empty, coap.InternalServerError},

		//5xx
		{"fasthttp.StatusInternalServerError", fasthttp.StatusInternalServerError, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusNotImplemented", fasthttp.StatusNotImplemented, coap.Empty, coap.NotImplemented},
		{"fasthttp.StatusBadGateway", fasthttp.StatusBadGateway, coap.Empty, coap.BadGateway},
		{"fasthttp.StatusServiceUnavailable", fasthttp.StatusServiceUnavailable, coap.Empty, coap.ServiceUnavailable},
		{"fasthttp.StatusGatewayTimeout", fasthttp.StatusGatewayTimeout, coap.Empty, coap.GatewayTimeout},
		{"fasthttp.StatusHTTPVersionNotSupported", fasthttp.StatusHTTPVersionNotSupported, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusVariantAlsoNegotiates", fasthttp.StatusVariantAlsoNegotiates, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusInsufficientStorage", fasthttp.StatusInsufficientStorage, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusLoopDetected", fasthttp.StatusLoopDetected, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusNotExtended", fasthttp.StatusNotExtended, coap.Empty, coap.InternalServerError},
		{"fasthttp.StatusNetworkAuthenticationRequired", fasthttp.StatusNetworkAuthenticationRequired, coap.Empty, coap.InternalServerError},
	}
	for _, e := range tbl {
		testCode := func(t *testing.T) {
			code := httpCode2CoapCode(e.inStatusCode, e.inCoapCode)
			if e.out != code {
				t.Errorf("Unexpected code(%v) returned, expected %v", code, e.out)
			}
		}
		t.Run(e.name, testCode)
	}
}
