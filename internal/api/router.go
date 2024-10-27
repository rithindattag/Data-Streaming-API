package api

import (
	"github.com/valyala/fasthttp"
	"github.com/rithindattag/realtime-streaming-api/pkg/auth"
	"strings"
)

func NewRouter(h *Handlers) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// API Key authentication
		if !auth.ValidateAPIKey(string(ctx.Request.Header.Peek("X-API-Key"))) {
			ctx.Error("Invalid API key", fasthttp.StatusUnauthorized)
			return
		}

		path := string(ctx.Path())
		switch {
		case path == "/stream/start":
			h.StartStream(ctx)
		case strings.HasPrefix(path, "/stream/") && strings.HasSuffix(path, "/send"):
			h.SendData(ctx)
		case strings.HasPrefix(path, "/stream/") && strings.HasSuffix(path, "/results"):
			h.StreamResults(ctx)
		default:
			ctx.Error("Not found", fasthttp.StatusNotFound)
		}
	}
}
