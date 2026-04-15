package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	ag2mocks "github.com/my-devstack/mydevstack-proxy/mocks/ports"
	"github.com/stretchr/testify/assert"
)

func TestNewAPIGatewayV2Adapter(t *testing.T) {
	adapter := NewAPIGatewayV2Adapter(aws.Config{Region: "us-east-1"}, "http://localhost:4566")
	assert.NotNil(t, adapter)
	assert.IsType(t, &APIGatewayV2Adapter{}, adapter)
}

func TestAPIGatewayV2Adapter_GetApis(t *testing.T) {
	mockClient := ag2mocks.NewAPIGatewayV2ClientPort(t)
	ctx := context.Background()
	input := &apigatewayv2.GetApisInput{}
	expectedOutput := &apigatewayv2.GetApisOutput{Items: []types.Api{{ApiId: aws.String("api-123")}}}

	mockClient.EXPECT().GetApis(ctx, input).Return(expectedOutput, nil)
	adapter := &APIGatewayV2Adapter{client: mockClient}

	output, err := adapter.GetApis(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestAPIGatewayV2Adapter_CreateApi(t *testing.T) {
	mockClient := ag2mocks.NewAPIGatewayV2ClientPort(t)
	ctx := context.Background()
	input := &apigatewayv2.CreateApiInput{Name: aws.String("test-api")}
	expectedOutput := &apigatewayv2.CreateApiOutput{ApiId: aws.String("api-123")}

	mockClient.EXPECT().CreateApi(ctx, input).Return(expectedOutput, nil)
	adapter := &APIGatewayV2Adapter{client: mockClient}

	output, err := adapter.CreateApi(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestAPIGatewayV2Adapter_DeleteApi(t *testing.T) {
	mockClient := ag2mocks.NewAPIGatewayV2ClientPort(t)
	ctx := context.Background()
	input := &apigatewayv2.DeleteApiInput{ApiId: aws.String("api-123")}
	expectedOutput := &apigatewayv2.DeleteApiOutput{}

	mockClient.EXPECT().DeleteApi(ctx, input).Return(expectedOutput, nil)
	adapter := &APIGatewayV2Adapter{client: mockClient}

	output, err := adapter.DeleteApi(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestAPIGatewayV2Adapter_GetApi(t *testing.T) {
	mockClient := ag2mocks.NewAPIGatewayV2ClientPort(t)
	ctx := context.Background()
	input := &apigatewayv2.GetApiInput{ApiId: aws.String("api-123")}
	expectedOutput := &apigatewayv2.GetApiOutput{ApiId: aws.String("api-123")}

	mockClient.EXPECT().GetApi(ctx, input).Return(expectedOutput, nil)
	adapter := &APIGatewayV2Adapter{client: mockClient}

	output, err := adapter.GetApi(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestAPIGatewayV2Adapter_GetRoutes(t *testing.T) {
	mockClient := ag2mocks.NewAPIGatewayV2ClientPort(t)
	ctx := context.Background()
	input := &apigatewayv2.GetRoutesInput{ApiId: aws.String("api-123")}
	expectedOutput := &apigatewayv2.GetRoutesOutput{}

	mockClient.EXPECT().GetRoutes(ctx, input).Return(expectedOutput, nil)
	adapter := &APIGatewayV2Adapter{client: mockClient}

	output, err := adapter.GetRoutes(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestAPIGatewayV2Adapter_CreateRoute(t *testing.T) {
	mockClient := ag2mocks.NewAPIGatewayV2ClientPort(t)
	ctx := context.Background()
	input := &apigatewayv2.CreateRouteInput{ApiId: aws.String("api-123")}
	expectedOutput := &apigatewayv2.CreateRouteOutput{RouteId: aws.String("route-123")}

	mockClient.EXPECT().CreateRoute(ctx, input).Return(expectedOutput, nil)
	adapter := &APIGatewayV2Adapter{client: mockClient}

	output, err := adapter.CreateRoute(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestAPIGatewayV2Adapter_DeleteRoute(t *testing.T) {
	mockClient := ag2mocks.NewAPIGatewayV2ClientPort(t)
	ctx := context.Background()
	input := &apigatewayv2.DeleteRouteInput{ApiId: aws.String("api-123"), RouteId: aws.String("route-123")}
	expectedOutput := &apigatewayv2.DeleteRouteOutput{}

	mockClient.EXPECT().DeleteRoute(ctx, input).Return(expectedOutput, nil)
	adapter := &APIGatewayV2Adapter{client: mockClient}

	output, err := adapter.DeleteRoute(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestAPIGatewayV2Adapter_GetIntegrations(t *testing.T) {
	mockClient := ag2mocks.NewAPIGatewayV2ClientPort(t)
	ctx := context.Background()
	input := &apigatewayv2.GetIntegrationsInput{ApiId: aws.String("api-123")}
	expectedOutput := &apigatewayv2.GetIntegrationsOutput{}

	mockClient.EXPECT().GetIntegrations(ctx, input).Return(expectedOutput, nil)
	adapter := &APIGatewayV2Adapter{client: mockClient}

	output, err := adapter.GetIntegrations(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestAPIGatewayV2Adapter_CreateIntegration(t *testing.T) {
	mockClient := ag2mocks.NewAPIGatewayV2ClientPort(t)
	ctx := context.Background()
	input := &apigatewayv2.CreateIntegrationInput{ApiId: aws.String("api-123")}
	expectedOutput := &apigatewayv2.CreateIntegrationOutput{IntegrationId: aws.String("int-123")}

	mockClient.EXPECT().CreateIntegration(ctx, input).Return(expectedOutput, nil)
	adapter := &APIGatewayV2Adapter{client: mockClient}

	output, err := adapter.CreateIntegration(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}
