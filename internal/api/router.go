package api

import (
	"github.com/valyala/fasthttp"
)

func NewRouter(h *Handlers) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/stream/start":
			h.StartStream(ctx)
		case "/stream/{stream_id}/send":
			h.SendData(ctx)
		case "/stream/{stream_id}/results":
			h.StreamResults(ctx)
		default:
			ctx.Error("Not found", fasthttp.StatusNotFound)
		}
	}
}
