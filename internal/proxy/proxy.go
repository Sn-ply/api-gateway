package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"

	"go.uber.org/zap"
)

type ReverseProxy struct {
	userProxy         *httputil.ReverseProxy
	postProxy         *httputil.ReverseProxy
	relationProxy     *httputil.ReverseProxy
	likeProxy         *httputil.ReverseProxy
	notificationProxy *httputil.ReverseProxy
	messageProxy      *httputil.ReverseProxy
	log               *zap.Logger
}

func New(userServiceURL, postServiceURL, relationServiceURL, likeServiceURL, notificationServiceURL, messageServiceURL string, log *zap.Logger) (*ReverseProxy, error) {
	userURL, err := url.Parse(userServiceURL)
	if err != nil {
		return nil, err
	}
	postURL, err := url.Parse(postServiceURL)
	if err != nil {
		return nil, err
	}
	relationURL, err := url.Parse(relationServiceURL)
	if err != nil {
		return nil, err
	}
	likeURL, err := url.Parse(likeServiceURL)
	if err != nil {
		return nil, err
	}
	notificationURL, err := url.Parse(notificationServiceURL)
	if err != nil {
		return nil, err
	}
	messageURL, err := url.Parse(messageServiceURL)
	if err != nil {
		return nil, err
	}

	return &ReverseProxy{
		userProxy:         newProxy(userURL, log),
		postProxy:         newProxy(postURL, log),
		relationProxy:     newProxy(relationURL, log),
		likeProxy:         newProxy(likeURL, log),
		notificationProxy: newProxy(notificationURL, log),
		messageProxy:      newProxy(messageURL, log),
		log:               log,
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

func (p *ReverseProxy) RelationService() http.Handler {
	return p.relationProxy
}

func (p *ReverseProxy) LikeService() http.Handler {
	return p.likeProxy
}

func (p *ReverseProxy) NotificationService() http.Handler {
	return p.notificationProxy
}

func (p *ReverseProxy) MessageService() http.Handler {
	return p.messageProxy
}
