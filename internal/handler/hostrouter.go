package handler

import (
	"errors"
	"net/http"
	"strings"
)

type HostRouter struct {
	hostRoutes map[string]http.Handler
}

func NewHostRouter(defaultHandler http.Handler) *HostRouter {
	return &HostRouter{
		hostRoutes: map[string]http.Handler{"*": defaultHandler},
	}
}

func (h *HostRouter) SetHostRoute(host string, handler http.Handler) error {
	if host == "*" {
		return errors.New("error cannot overide default(*) router")
	}
	_, ok := h.hostRoutes[strings.ToLower(host)]
	if ok {
		return errors.New("error cannot override previously specified hostname")
	}

	h.hostRoutes[strings.ToLower(host)] = handler
	return nil
}

func (hr *HostRouter) Handler() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		router, ok := hr.hostRoutes[strings.ToLower(req.Host)]
		if ok {
			router.ServeHTTP(res, req)
			return
		}

		// fallback to the default router
		defaultRouter := hr.hostRoutes["*"]
		defaultRouter.ServeHTTP(res, req)
	})
}
