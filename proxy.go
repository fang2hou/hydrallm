package main

import (
	"net/http"
	"net/http/httputil"

	"github.com/charmbracelet/log"
)

func newProxy(listener *Listener, cfg *Config, logger *log.Logger) *httputil.ReverseProxy {
	transport := newRetryTransport(
		listener.ResolvedModels,
		cfg.Providers,
		cfg.Retry,
		cfg.Log,
		logger,
	)

	return &httputil.ReverseProxy{
		Rewrite: func(req *httputil.ProxyRequest) {
			logger.Debug(
				"incoming request",
				"method",
				req.In.Method,
				"path",
				req.In.URL.Path,
				"host",
				req.In.Host,
			)
		},
		Transport:     transport,
		FlushInterval: -1, // Flush immediately for streaming
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("proxy error", "error", err, "path", r.URL.Path, "method", r.Method)
			http.Error(w, "proxy error: "+err.Error(), http.StatusBadGateway)
		},
	}
}
