package bootstrap

import (
	httphandlers "github.com/my-devstack/mydevstack-proxy/internal/adapters/http"
	"github.com/my-devstack/mydevstack-proxy/internal/application"
	configloader "github.com/my-devstack/mydevstack-proxy/internal/config"
	"github.com/my-devstack/mydevstack-proxy/internal/ports"
)

type Container struct {
	Config  *configloader.Config
	Service ports.ProxyService
	Handler *httphandlers.ProxyHandler
}

func NewContainer(cfg *configloader.Config) (*Container, error) {
	svc, err := application.NewProxyService(cfg)
	if err != nil {
		return nil, err
	}

	handler := httphandlers.NewProxyHandler(svc)

	return &Container{
		Config:  cfg,
		Service: svc,
		Handler: handler,
	}, nil
}
