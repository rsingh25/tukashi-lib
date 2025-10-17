package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/rsingh25/tukashi-lib/lambda/albproxy/core"
	"github.com/rsingh25/tukashi-lib/util"

	"github.com/aws/aws-lambda-go/events"
)

var appLog *slog.Logger

func init() {
	appLog = util.Logger.With("package", "httpadapter")
}

type HandlerAdapterALB struct {
	core.RequestAccessorALB
	handler http.Handler
}

func NewALB(handler http.Handler) *HandlerAdapterALB {
	return &HandlerAdapterALB{
		handler: handler,
	}
}

// Proxy receives an ALB Target Group proxy event, transforms it into an http.Request
// object, and sends it to the http.HandlerFunc for routing.
// It returns a proxy response object generated from the http.ResponseWriter.
func (h *HandlerAdapterALB) Proxy(event events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	req, err := h.ProxyEventToHTTPRequest(event)
	return h.proxyInternal(req, err)
}

// ProxyWithContext receives context and an ALB proxy event,
// transforms them into an http.Request object, and sends it to the http.Handler for routing.
// It returns a proxy response object generated from the http.ResponseWriter.
func (h *HandlerAdapterALB) ProxyWithContext(ctx context.Context, event events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	appLog.Debug("Received ABL Request", "event", event)
	req, err := h.EventToRequestWithContext(ctx, event)
	if err != nil {
		appLog.Error("Could not convert proxy event to request", "event", event, "err", err)
	} else {
		appLog.Debug("Convered proxy event to request", "event", event, "header", req.Header, "method", req.Method, "URL", req.URL)

	}
	return h.proxyInternal(req, err)
}

func (h *HandlerAdapterALB) proxyInternal(req *http.Request, err error) (events.ALBTargetGroupResponse, error) {
	if err != nil {
		return core.GatewayTimeoutALB(), core.NewLoggedError("Could not convert proxy event to request: %v", err)
	}

	w := core.NewProxyResponseWriterALB()
	h.handler.ServeHTTP(http.ResponseWriter(w), req)

	resp, err := w.GetProxyResponse()
	if err != nil {
		appLog.Error("Error while generating proxy response", "err", err)
		return core.GatewayTimeoutALB(), core.NewLoggedError("Error while generating proxy response: %v", err)
	} else {
		appLog.Debug("Generated proxy response", "resp", resp)
	}

	return resp, nil
}
