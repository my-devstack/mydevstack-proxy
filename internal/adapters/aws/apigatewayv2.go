package aws

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/my-devstack/mydevstack-proxy/internal/ports"
)

type APIGatewayV2Adapter struct {
	client *apigatewayv2.Client
}

func NewAPIGatewayV2Adapter(awsCfg aws.Config, endpoint string) ports.APIGatewayV2Port {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	client := apigatewayv2.NewFromConfig(awsCfg, func(o *apigatewayv2.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.HTTPClient = httpClient
	})
	return &APIGatewayV2Adapter{client: client}
}

func (a *APIGatewayV2Adapter) GetApis(ctx context.Context, input *apigatewayv2.GetApisInput) (*apigatewayv2.GetApisOutput, error) {
	return a.client.GetApis(ctx, input)
}

func (a *APIGatewayV2Adapter) CreateApi(ctx context.Context, input *apigatewayv2.CreateApiInput) (*apigatewayv2.CreateApiOutput, error) {
	return a.client.CreateApi(ctx, input)
}

func (a *APIGatewayV2Adapter) DeleteApi(ctx context.Context, input *apigatewayv2.DeleteApiInput) (*apigatewayv2.DeleteApiOutput, error) {
	return a.client.DeleteApi(ctx, input)
}

func (a *APIGatewayV2Adapter) GetApi(ctx context.Context, input *apigatewayv2.GetApiInput) (*apigatewayv2.GetApiOutput, error) {
	return a.client.GetApi(ctx, input)
}

func (a *APIGatewayV2Adapter) GetRoutes(ctx context.Context, input *apigatewayv2.GetRoutesInput) (*apigatewayv2.GetRoutesOutput, error) {
	return a.client.GetRoutes(ctx, input)
}

func (a *APIGatewayV2Adapter) CreateRoute(ctx context.Context, input *apigatewayv2.CreateRouteInput) (*apigatewayv2.CreateRouteOutput, error) {
	return a.client.CreateRoute(ctx, input)
}

func (a *APIGatewayV2Adapter) DeleteRoute(ctx context.Context, input *apigatewayv2.DeleteRouteInput) (*apigatewayv2.DeleteRouteOutput, error) {
	return a.client.DeleteRoute(ctx, input)
}

func (a *APIGatewayV2Adapter) GetIntegrations(ctx context.Context, input *apigatewayv2.GetIntegrationsInput) (*apigatewayv2.GetIntegrationsOutput, error) {
	return a.client.GetIntegrations(ctx, input)
}

func (a *APIGatewayV2Adapter) CreateIntegration(ctx context.Context, input *apigatewayv2.CreateIntegrationInput) (*apigatewayv2.CreateIntegrationOutput, error) {
	return a.client.CreateIntegration(ctx, input)
}
