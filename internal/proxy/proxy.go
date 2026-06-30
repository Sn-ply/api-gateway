package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"

	"go.uber.org/zap"
)

type ReverseProxy struct {
	userProxy *httputil.ReverseProxy
	postProxy *httputil.ReverseProxy
	log       *zap.Logger
}

func New(userServiceURL, postServiceURL string, log *zap.Logger) (*ReverseProxy, error) {
	userURL, err := url.Parse(userServiceURL)
	if err != nil {
		return nil, err
	}
	postURL, err := url.Parse(postServiceURL)
	if err != nil {
		return nil, err
	}

	return &ReverseProxy{
		userProxy: newProxy(userURL, log),
		postProxy: newProxy(postURL, log),
		log:       log,
	}, nil
}

func newProxy(target *url.URL, log *zap.Logger) *httputil.ReverseProxy {
	p := httputil.NewSingleHostReverseProxy(target)
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("proxy error", zap.String("target", target.String()), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "upstream service unavailable"})
	}
	return p
}

func (p *ReverseProxy) UserService() http.Handler {
	return p.userProxy
}

func (p *ReverseProxy) PostService() http.Handler {
	return p.postProxy
}
