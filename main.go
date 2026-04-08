package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	apigwTypes "github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Config holds the proxy server configuration
type Config struct {
	Port           string
	AwsEndpoint    string
	AwsRegion      string
	AwsAccessKey   string
	AwsSecretKey   string
	ServicePattern string
}

func loadConfig() *Config {
	godotenv.Load()

	return &Config{
		Port:           getEnv("PROXY_PORT", "8081"),
		AwsEndpoint:    getEnv("AWS_ENDPOINT", "http://localhost:4550"),
		AwsRegion:      getEnv("AWS_REGION", "us-east-1"),
		AwsAccessKey:   getEnv("AWS_ACCESS_KEY", "test"),
		AwsSecretKey:   getEnv("AWS_SECRET_KEY", "test"),
		ServicePattern: getEnv("SERVICE_PATTERN", "root"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ProxyHandler handles proxying requests to AWS services using AWS SDK
type ProxyHandler struct {
	cfg            *Config
	secretsManager *secretsmanager.Client
	s3             *s3.Client
	lambda         *lambda.Client
	sqs            *sqs.Client
	sns            *sns.Client
	kms            *kms.Client
	dynamodb       *dynamodb.Client
	apigateway     *apigateway.Client
	apigatewayv2   *apigatewayv2.Client
	ssm            *ssm.Client
	iam            *iam.Client
	kinesis        *kinesis.Client
}

func NewProxyHandler(cfg *Config) *ProxyHandler {
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.AwsRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AwsAccessKey,
			cfg.AwsSecretKey,
			"",
		)),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}

	secretsClient := secretsmanager.NewFromConfig(awsCfg, func(o *secretsmanager.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
		o.UsePathStyle = true
	})

	lambdaClient := lambda.NewFromConfig(awsCfg, func(o *lambda.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	sqsClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	snsClient := sns.NewFromConfig(awsCfg, func(o *sns.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	kmsClient := kms.NewFromConfig(awsCfg, func(o *kms.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	dynamoDBClient := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	apiGatewayClient := apigateway.NewFromConfig(awsCfg, func(o *apigateway.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	apiGatewayV2Client := apigatewayv2.NewFromConfig(awsCfg, func(o *apigatewayv2.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	ssmClient := ssm.NewFromConfig(awsCfg, func(o *ssm.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	iamClient := iam.NewFromConfig(awsCfg, func(o *iam.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	kinesisClient := kinesis.NewFromConfig(awsCfg, func(o *kinesis.Options) {
		o.BaseEndpoint = aws.String(cfg.AwsEndpoint)
		o.HTTPClient = httpClient
	})

	return &ProxyHandler{
		cfg:            cfg,
		secretsManager: secretsClient,
		s3:             s3Client,
		lambda:         lambdaClient,
		sqs:            sqsClient,
		sns:            snsClient,
		kms:            kmsClient,
		dynamodb:       dynamoDBClient,
		apigateway:     apiGatewayClient,
		apigatewayv2:   apiGatewayV2Client,
		ssm:            ssmClient,
		iam:            iamClient,
		kinesis:        kinesisClient,
	}
}

func (h *ProxyHandler) serviceRouter(c *gin.Context) {
	service := c.Param("service")

	switch service {
	case "secretsmanager":
		h.handleSecretsManager(c)
	case "s3":
		h.handleS3(c)
	case "lambda":
		h.handleLambda(c)
	case "sqs":
		h.handleSQS(c)
	case "sns":
		h.handleSNS(c)
	case "kms":
		h.handleKMS(c)
	case "dynamodb":
		h.handleDynamoDB(c)
	case "apigateway":
		h.handleAPIGateway(c)
	case "ssm":
		h.handleSSM(c)
	case "iam":
		h.handleIAM(c)
	case "kinesis":
		h.handleKinesis(c)
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not supported: " + service})
	}
}

// ==================== S3 HANDLERS ====================

func (h *ProxyHandler) handleS3(c *gin.Context) {
	path := c.Param("path")
	xAmzTarget := c.GetHeader("X-Amz-Target")

	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}

	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "ListBuckets"):
		h.listBuckets(ctx, c)
	case strings.Contains(xAmzTarget, "ListObjectsV2"):
		h.listObjectsV2(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetObject"):
		h.getObject(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "PutObject"):
		h.putObject(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteObject"):
		h.deleteObject(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteBucket"):
		h.deleteBucket(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "HeadBucket"):
		h.headBucket(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "HeadObject"):
		h.headObject(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateBucket"):
		h.createBucket(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown S3 action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) listBuckets(ctx context.Context, c *gin.Context) {
	result, err := h.s3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list buckets", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listObjectsV2(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &s3.ListObjectsV2Input{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.s3.ListObjectsV2(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list objects", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getObject(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &s3.GetObjectInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.s3.GetObject(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get object", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) putObject(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &s3.PutObjectInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.s3.PutObject(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to put object", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteObject(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &s3.DeleteObjectInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.s3.DeleteObject(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete object", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteBucket(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &s3.DeleteBucketInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.s3.DeleteBucket(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete bucket", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) headBucket(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &s3.HeadBucketInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	_, err := h.s3.HeadBucket(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to head bucket", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

func (h *ProxyHandler) headObject(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &s3.HeadObjectInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.s3.HeadObject(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to head object", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createBucket(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &s3.CreateBucketInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.s3.CreateBucket(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create bucket", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== LAMBDA HANDLERS ====================

func (h *ProxyHandler) handleLambda(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "ListFunctions"):
		h.listFunctions(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateFunction"):
		h.createFunction(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetFunction"):
		h.getFunction(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteFunction"):
		h.deleteFunction(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "Invoke"):
		h.invokeFunction(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateFunctionConfiguration"):
		h.updateFunctionConfiguration(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateFunctionCode"):
		h.updateFunctionCode(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetFunctionConfiguration"):
		h.getFunctionConfiguration(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown Lambda action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) listFunctions(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &lambda.ListFunctionsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.lambda.ListFunctions(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list functions", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createFunction(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &lambda.CreateFunctionInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.lambda.CreateFunction(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create function", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getFunction(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &lambda.GetFunctionInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.lambda.GetFunction(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get function", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteFunction(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &lambda.DeleteFunctionInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.lambda.DeleteFunction(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete function", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) invokeFunction(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &lambda.InvokeInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.lambda.Invoke(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to invoke function", err)
		return
	}

	response := map[string]interface{}{
		"StatusCode": result.StatusCode,
	}
	if result.FunctionError != nil {
		response["FunctionError"] = *result.FunctionError
		c.Header("X-Amz-Function-Error", *result.FunctionError)
	}
	if len(result.Payload) > 0 {
		encoded := base64.StdEncoding.EncodeToString(result.Payload)
		response["Payload"] = encoded
	}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, response)
}

func (h *ProxyHandler) updateFunctionConfiguration(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &lambda.UpdateFunctionConfigurationInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.lambda.UpdateFunctionConfiguration(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update function configuration", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) updateFunctionCode(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &lambda.UpdateFunctionCodeInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.lambda.UpdateFunctionCode(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update function code", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getFunctionConfiguration(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &lambda.GetFunctionConfigurationInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.lambda.GetFunctionConfiguration(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get function configuration", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== SECRETS MANAGER HANDLERS ====================

func (h *ProxyHandler) handleSecretsManager(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "ListSecrets"):
		h.listSecrets(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateSecret"):
		h.createSecret(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetSecretValue"):
		h.getSecretValue(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "PutSecretValue"):
		h.putSecretValue(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteSecret"):
		h.deleteSecret(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DescribeSecret"):
		h.describeSecret(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateSecret"):
		h.updateSecret(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "RestoreSecret"):
		h.restoreSecret(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "RotateSecret"):
		h.rotateSecret(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetRandomPassword"):
		h.getRandomPassword(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown Secrets Manager action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) listSecrets(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.ListSecretsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.ListSecrets(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list secrets", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createSecret(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.CreateSecretInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.CreateSecret(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create secret", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getSecretValue(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.GetSecretValueInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.GetSecretValue(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get secret value", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) putSecretValue(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.PutSecretValueInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.PutSecretValue(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to put secret value", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteSecret(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.DeleteSecretInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.DeleteSecret(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete secret", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) describeSecret(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.DescribeSecretInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.DescribeSecret(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to describe secret", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) updateSecret(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.UpdateSecretInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.UpdateSecret(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update secret", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) restoreSecret(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.RestoreSecretInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.RestoreSecret(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to restore secret", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) rotateSecret(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.RotateSecretInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.RotateSecret(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to rotate secret", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getRandomPassword(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &secretsmanager.GetRandomPasswordInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.secretsManager.GetRandomPassword(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get random password", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== SQS HANDLERS ====================

func (h *ProxyHandler) handleSQS(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "ListQueues"):
		h.listQueues(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateQueue"):
		h.createQueue(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteQueue"):
		h.deleteQueue(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetQueueUrl"):
		h.getQueueUrl(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "SendMessage"):
		h.sendMessage(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ReceiveMessage"):
		h.receiveMessage(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteMessage"):
		h.deleteMessage(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "PurgeQueue"):
		h.purgeQueue(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetQueueAttributes"):
		h.getQueueAttributes(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "SetQueueAttributes"):
		h.setQueueAttributes(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown SQS action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) listQueues(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.ListQueuesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.ListQueues(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list queues", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createQueue(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.CreateQueueInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.CreateQueue(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create queue", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteQueue(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.DeleteQueueInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.DeleteQueue(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete queue", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getQueueUrl(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.GetQueueUrlInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.GetQueueUrl(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get queue URL", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) sendMessage(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.SendMessageInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.SendMessage(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to send message", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) receiveMessage(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.ReceiveMessageInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.ReceiveMessage(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to receive message", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteMessage(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.DeleteMessageInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.DeleteMessage(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete message", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) purgeQueue(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.PurgeQueueInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.PurgeQueue(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to purge queue", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getQueueAttributes(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.GetQueueAttributesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.GetQueueAttributes(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get queue attributes", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) setQueueAttributes(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sqs.SetQueueAttributesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sqs.SetQueueAttributes(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to set queue attributes", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== SNS HANDLERS ====================

func (h *ProxyHandler) handleSNS(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "ListTopics"):
		h.listTopics(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateTopic"):
		h.createTopic(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteTopic"):
		h.deleteTopic(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "Subscribe"):
		h.subscribe(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "Unsubscribe"):
		h.unsubscribe(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListSubscriptions"):
		h.listSubscriptions(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListSubscriptionsByTopic"):
		h.listSubscriptionsByTopic(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "Publish"):
		h.publish(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown SNS action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) listTopics(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sns.ListTopicsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sns.ListTopics(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list topics", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createTopic(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sns.CreateTopicInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sns.CreateTopic(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create topic", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteTopic(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sns.DeleteTopicInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sns.DeleteTopic(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete topic", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) subscribe(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sns.SubscribeInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sns.Subscribe(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to subscribe", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) unsubscribe(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sns.UnsubscribeInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sns.Unsubscribe(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to unsubscribe", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listSubscriptions(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sns.ListSubscriptionsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sns.ListSubscriptions(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list subscriptions", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listSubscriptionsByTopic(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sns.ListSubscriptionsByTopicInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sns.ListSubscriptionsByTopic(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list subscriptions by topic", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) publish(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &sns.PublishInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.sns.Publish(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to publish", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== KMS HANDLERS ====================

func (h *ProxyHandler) handleKMS(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "ListKeys"):
		h.listKeys(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateKey"):
		h.createKey(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteAlias"):
		h.deleteAlias(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DescribeKey"):
		h.describeKey(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "Encrypt"):
		h.encrypt(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "Decrypt"):
		h.decrypt(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GenerateDataKey"):
		h.generateDataKey(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GenerateRandom"):
		h.generateRandom(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown KMS action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) listKeys(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kms.ListKeysInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kms.ListKeys(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list keys", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createKey(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kms.CreateKeyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kms.CreateKey(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create key", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteAlias(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kms.DeleteAliasInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kms.DeleteAlias(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete alias", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) describeKey(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kms.DescribeKeyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kms.DescribeKey(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to describe key", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) encrypt(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kms.EncryptInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kms.Encrypt(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to encrypt", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) decrypt(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kms.DecryptInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kms.Decrypt(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to decrypt", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) generateDataKey(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kms.GenerateDataKeyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kms.GenerateDataKey(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to generate data key", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) generateRandom(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kms.GenerateRandomInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kms.GenerateRandom(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to generate random", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== DYNAMODB HANDLERS ====================

func (h *ProxyHandler) handleDynamoDB(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "ListTables"):
		h.listTables(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateTable"):
		h.createTable(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DescribeTable"):
		h.describeTable(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteTable"):
		h.deleteTable(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateTable"):
		h.updateTable(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "PutItem"):
		h.putItem(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetItem"):
		h.getItem(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteItem"):
		h.deleteItem(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateItem"):
		h.updateItem(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "Query"):
		h.query(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "Scan"):
		h.scan(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "BatchWriteItem"):
		h.batchWriteItem(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "BatchGetItem"):
		h.batchGetItem(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DescribeTimeToLive"):
		h.describeTimeToLive(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateTimeToLive"):
		h.updateTimeToLive(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown DynamoDB action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) listTables(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &dynamodb.ListTablesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.dynamodb.ListTables(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list tables", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createTable(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &dynamodb.CreateTableInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.dynamodb.CreateTable(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create table", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) describeTable(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &dynamodb.DescribeTableInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.dynamodb.DescribeTable(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to describe table", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteTable(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &dynamodb.DeleteTableInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.dynamodb.DeleteTable(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete table", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) updateTable(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &dynamodb.UpdateTableInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.dynamodb.UpdateTable(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update table", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) putItem(ctx context.Context, c *gin.Context, bodyBytes []byte) {

	// Parse the body into a generic map first
	var rawBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawBody); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	input := &dynamodb.PutItemInput{}

	// Extract TableName
	if tableName, ok := rawBody["TableName"].(string); ok {
		input.TableName = aws.String(tableName)
	}

	// Extract and unmarshal Item
	if itemData, ok := rawBody["Item"].(map[string]interface{}); ok {
		item := make(map[string]types.AttributeValue)
		for key, value := range itemData {
			attrValue := convertToAttributeValue(value)
			if attrValue != nil {
				item[key] = attrValue
			}
		}
		input.Item = item
	}

	// Extract other optional fields
	if val, ok := rawBody["ConditionExpression"].(string); ok {
		input.ConditionExpression = aws.String(val)
	}
	if val, ok := rawBody["ReturnValues"].(string); ok {
		input.ReturnValues = types.ReturnValue(val)
	}

	result, err := h.dynamodb.PutItem(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to put item", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// convertToAttributeValue converts a JSON value to a DynamoDB AttributeValue
// Handles multiple formats: {"S": "value"} or {"M": {"Value": {...}}}
func convertToAttributeValue(value interface{}) types.AttributeValue {
	if value == nil {
		return &types.AttributeValueMemberNULL{Value: true}
	}

	switch v := value.(type) {
	case string:
		return &types.AttributeValueMemberS{Value: v}
	case float64:
		return &types.AttributeValueMemberN{Value: strconv.FormatFloat(v, 'f', -1, 64)}
	case bool:
		return &types.AttributeValueMemberBOOL{Value: v}
	case map[string]interface{}:
		// Check for wrapper format {"M": {"Value": {...}}}
		if m, ok := v["M"].(map[string]interface{}); ok {
			if innerValue, ok := m["Value"]; ok {
				return convertToAttributeValue(innerValue)
			}
		}

		// Check for DynamoDB attribute format like {"S": "value"}
		if s, ok := v["S"].(string); ok {
			return &types.AttributeValueMemberS{Value: s}
		}
		if n, ok := v["N"].(string); ok {
			return &types.AttributeValueMemberN{Value: n}
		}
		if b, ok := v["B"].(string); ok {
			decoded, _ := base64.StdEncoding.DecodeString(b)
			return &types.AttributeValueMemberB{Value: decoded}
		}
		if _, ok := v["BOOL"].(bool); ok {
			return &types.AttributeValueMemberBOOL{Value: v["BOOL"].(bool)}
		}
		if _, ok := v["NULL"].(bool); ok {
			return &types.AttributeValueMemberNULL{Value: true}
		}
		if l, ok := v["L"].([]interface{}); ok {
			list := make([]types.AttributeValue, len(l))
			for i, elem := range l {
				list[i] = convertToAttributeValue(elem)
			}
			return &types.AttributeValueMemberL{Value: list}
		}
		if m, ok := v["M"].(map[string]interface{}); ok {
			memberMap := make(map[string]types.AttributeValue)
			for mk, mv := range m {
				memberMap[mk] = convertToAttributeValue(mv)
			}
			return &types.AttributeValueMemberM{Value: memberMap}
		}
		if ss, ok := v["SS"].([]interface{}); ok {
			strSet := make([]string, len(ss))
			for i, s := range ss {
				if str, ok := s.(string); ok {
					strSet[i] = str
				}
			}
			return &types.AttributeValueMemberSS{Value: strSet}
		}
		if ns, ok := v["NS"].([]interface{}); ok {
			numSet := make([]string, len(ns))
			for i, n := range ns {
				if num, ok := n.(string); ok {
					numSet[i] = num
				}
			}
			return &types.AttributeValueMemberNS{Value: numSet}
		}
		// Fallback: treat as map
		memberMap := make(map[string]types.AttributeValue)
		for mk, mv := range v {
			memberMap[mk] = convertToAttributeValue(mv)
		}
		return &types.AttributeValueMemberM{Value: memberMap}
	case []interface{}:
		list := make([]types.AttributeValue, len(v))
		for i, elem := range v {
			list[i] = convertToAttributeValue(elem)
		}
		return &types.AttributeValueMemberL{Value: list}
	default:
		return &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", v)}
	}
}

func (h *ProxyHandler) getItem(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	// Parse the body into a generic map first
	var rawBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawBody); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	input := &dynamodb.GetItemInput{}

	// Extract TableName
	if tableName, ok := rawBody["TableName"].(string); ok {
		input.TableName = aws.String(tableName)
	}

	// Extract and unmarshal Key
	if keyData, ok := rawBody["Key"].(map[string]interface{}); ok {
		key := make(map[string]types.AttributeValue)
		for k, value := range keyData {
			attrValue := convertToAttributeValue(value)
			if attrValue != nil {
				key[k] = attrValue
			}
		}
		input.Key = key
	}

	// Extract other optional fields
	if val, ok := rawBody["ConsistentRead"].(bool); ok {
		input.ConsistentRead = aws.Bool(val)
	}
	if val, ok := rawBody["ProjectionExpression"].(string); ok {
		input.ProjectionExpression = aws.String(val)
	}

	result, err := h.dynamodb.GetItem(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get item", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteItem(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	// Parse the body into a generic map first
	var rawBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawBody); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	input := &dynamodb.DeleteItemInput{}

	// Extract TableName
	if tableName, ok := rawBody["TableName"].(string); ok {
		input.TableName = aws.String(tableName)
	}

	// Extract and unmarshal Key
	if keyData, ok := rawBody["Key"].(map[string]interface{}); ok {
		key := make(map[string]types.AttributeValue)
		for k, value := range keyData {
			attrValue := convertToAttributeValue(value)
			if attrValue != nil {
				key[k] = attrValue
			}
		}
		input.Key = key
	}

	// Extract other optional fields
	if val, ok := rawBody["ConditionExpression"].(string); ok {
		input.ConditionExpression = aws.String(val)
	}
	if val, ok := rawBody["ReturnValues"].(string); ok {
		input.ReturnValues = types.ReturnValue(val)
	}

	result, err := h.dynamodb.DeleteItem(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete item", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) updateItem(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	log.Printf("UpdateItem request body: %s", string(bodyBytes))

	// Parse the body into a generic map first
	var rawBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawBody); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	input := &dynamodb.UpdateItemInput{}

	// Extract TableName
	if tableName, ok := rawBody["TableName"].(string); ok {
		input.TableName = aws.String(tableName)
	}

	// Extract and unmarshal Key
	if keyData, ok := rawBody["Key"].(map[string]interface{}); ok {
		key := make(map[string]types.AttributeValue)
		for k, value := range keyData {
			attrValue := convertToAttributeValue(value)
			if attrValue != nil {
				key[k] = attrValue
			}
		}
		input.Key = key
	}

	// Extract optional fields
	if val, ok := rawBody["UpdateExpression"].(string); ok {
		input.UpdateExpression = aws.String(val)
	}
	if val, ok := rawBody["ConditionExpression"].(string); ok {
		input.ConditionExpression = aws.String(val)
	}
	if val, ok := rawBody["ReturnValues"].(string); ok {
		input.ReturnValues = types.ReturnValue(val)
	}

	result, err := h.dynamodb.UpdateItem(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update item", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) query(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	// Parse the body into a generic map first
	var rawBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawBody); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	input := &dynamodb.QueryInput{}

	// Extract TableName
	if tableName, ok := rawBody["TableName"].(string); ok {
		input.TableName = aws.String(tableName)
	}

	// Extract optional fields
	if val, ok := rawBody["KeyConditionExpression"].(string); ok {
		input.KeyConditionExpression = aws.String(val)
	}
	if val, ok := rawBody["FilterExpression"].(string); ok {
		input.FilterExpression = aws.String(val)
	}
	if val, ok := rawBody["ProjectionExpression"].(string); ok {
		input.ProjectionExpression = aws.String(val)
	}
	if val, ok := rawBody["Limit"].(float64); ok {
		input.Limit = aws.Int32(int32(val))
	}
	if val, ok := rawBody["ScanIndexForward"].(bool); ok {
		input.ScanIndexForward = aws.Bool(val)
	}
	if val, ok := rawBody["ExclusiveStartKey"].(map[string]interface{}); ok {
		input.ExclusiveStartKey = convertMapToAttributeValue(val)
	}

	result, err := h.dynamodb.Query(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to query", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) scan(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	// Parse the body into a generic map first
	var rawBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawBody); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	input := &dynamodb.ScanInput{}

	// Extract TableName
	if tableName, ok := rawBody["TableName"].(string); ok {
		input.TableName = aws.String(tableName)
	}

	// Extract optional fields
	if val, ok := rawBody["Limit"].(float64); ok {
		input.Limit = aws.Int32(int32(val))
	}
	if val, ok := rawBody["FilterExpression"].(string); ok {
		input.FilterExpression = aws.String(val)
	}
	if val, ok := rawBody["ProjectionExpression"].(string); ok {
		input.ProjectionExpression = aws.String(val)
	}
	if val, ok := rawBody["ExclusiveStartKey"].(map[string]interface{}); ok {
		input.ExclusiveStartKey = convertMapToAttributeValue(val)
	}

	result, err := h.dynamodb.Scan(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to scan", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// convertMapToAttributeValue converts a map to DynamoDB AttributeValue map
func convertMapToAttributeValue(data map[string]interface{}) map[string]types.AttributeValue {
	if data == nil {
		return nil
	}
	result := make(map[string]types.AttributeValue)
	for k, v := range data {
		result[k] = convertToAttributeValue(v)
	}
	return result
}

func (h *ProxyHandler) batchWriteItem(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &dynamodb.BatchWriteItemInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.dynamodb.BatchWriteItem(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to batch write", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) batchGetItem(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &dynamodb.BatchGetItemInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.dynamodb.BatchGetItem(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to batch get", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) describeTimeToLive(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &dynamodb.DescribeTimeToLiveInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.dynamodb.DescribeTimeToLive(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to describe TTL", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) updateTimeToLive(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &dynamodb.UpdateTimeToLiveInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.dynamodb.UpdateTimeToLive(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update TTL", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== API GATEWAY HANDLERS ====================

func (h *ProxyHandler) handleAPIGateway(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "GetRestApis"):
		h.getRestApis(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateRestApi"):
		h.createRestApi(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteRestApi"):
		h.deleteRestApi(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetRestApi"):
		h.getRestApi(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateRestApi"):
		h.updateRestApi(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetResources"):
		h.getResources(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetResource"):
		h.getResource(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateResource"):
		h.createResource(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteResource"):
		h.deleteResource(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "PutMethod"):
		h.putMethod(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetMethod"):
		h.getMethod(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteMethod"):
		h.deleteMethod(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "PutIntegration"):
		h.putIntegration(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetIntegration"):
		h.getIntegration(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteIntegration"):
		h.deleteIntegration(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateDeployment"):
		h.createDeployment(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteDeployment"):
		h.deleteDeployment(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateStage"):
		h.createStage(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetStages"):
		h.getStages(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateStage"):
		h.updateStage(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteStage"):
		h.deleteStage(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetApis"):
		h.getApis(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateApi"):
		h.createApi(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteApi"):
		h.deleteApi(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetApi"):
		h.getApi(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetRoutes"):
		h.getRoutes(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateRoute"):
		h.createRoute(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteRoute"):
		h.deleteRoute(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetIntegrations"):
		h.getIntegrationsV2(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateIntegration"):
		h.createIntegrationV2(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ImportRestApi"):
		h.importRestApi(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown API Gateway action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) getRestApis(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.GetRestApisInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.GetRestApis(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get REST APIs", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createRestApi(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.CreateRestApiInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.CreateRestApi(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create REST API", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) importRestApi(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	log.Printf("ImportRestApi received %d bytes", len(bodyBytes))

	// Check if body is JSON with "body" field (base64 encoded)
	var bodyData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyData); err == nil {
		if body, ok := bodyData["body"].(string); ok {
			// Decode base64
			decoded, err := base64.StdEncoding.DecodeString(body)
			if err != nil {
				log.Printf("ImportRestApi: base64 decode failed: %v", err)
				decoded = []byte(body)
			}
			log.Printf("ImportRestApi: Using base64 decoded body (%d bytes)", len(decoded))

			input := &apigateway.ImportRestApiInput{
				Body: decoded,
			}
			result, err := h.apigateway.ImportRestApi(ctx, input)
			if err != nil {
				log.Printf("ImportRestApi error (base64): %v", err)
				sendError(c, http.StatusInternalServerError, "Failed to import REST API", err)
				return
			}
			c.JSON(http.StatusOK, result)
			return
		}
	}

	// Check if body looks like Swagger JSON (starts with {)
	if len(bodyBytes) > 0 && bodyBytes[0] == byte('{') {
		log.Printf("ImportRestApi: Detected JSON Swagger spec")
		input := &apigateway.ImportRestApiInput{
			Body: bodyBytes,
		}
		result, err := h.apigateway.ImportRestApi(ctx, input)
		if err != nil {
			log.Printf("ImportRestApi error (JSON): %v", err)
			sendError(c, http.StatusInternalServerError, "Failed to import REST API", err)
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	// Try raw body
	log.Printf("ImportRestApi: Using raw body (%d bytes)", len(bodyBytes))
	input := &apigateway.ImportRestApiInput{
		Body: bodyBytes,
	}
	result, err := h.apigateway.ImportRestApi(ctx, input)
	if err != nil {
		log.Printf("ImportRestApi error (raw): %v", err)
		sendError(c, http.StatusInternalServerError, "Failed to import REST API", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteRestApi(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.DeleteRestApiInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.DeleteRestApi(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete REST API", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getRestApi(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	log.Printf("GetRestApi body: %s", string(bodyBytes))

	input := &apigateway.GetRestApiInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	log.Printf("GetRestApi input: RestApiId=%s", aws.ToString(input.RestApiId))

	result, err := h.apigateway.GetRestApi(ctx, input)
	if err != nil {
		log.Printf("GetRestApi error: %v", err)
		sendError(c, http.StatusInternalServerError, "Failed to get REST API", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) updateRestApi(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	log.Printf("UpdateRestApi body: %s", string(bodyBytes))

	var bodyData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyData); err != nil {
		log.Printf("UpdateRestApi: json unmarshal error: %v", err)
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	log.Printf("UpdateRestApi bodyData: %+v", bodyData)

	// Check if it's using AWS SDK format (array of patch operations) vs simple format (name/description fields)
	// The simple format from Vue has "name" and "description" as top-level fields
	// The AWS SDK format has "patchOperations" as an array
	if nameVal, hasName := bodyData["name"]; hasName {
		// Simple format: convert name/description to patch operations
		log.Printf("UpdateRestApi: using simple format")
		patchOperations := []apigwTypes.PatchOperation{}

		if name, ok := nameVal.(string); ok && name != "" {
			patchOperations = append(patchOperations, apigwTypes.PatchOperation{
				Op:    apigwTypes.OpReplace,
				Path:  aws.String("/name"),
				Value: aws.String(name),
			})
		}

		if desc, hasDesc := bodyData["description"].(string); hasDesc {
			patchOperations = append(patchOperations, apigwTypes.PatchOperation{
				Op:    apigwTypes.OpReplace,
				Path:  aws.String("/description"),
				Value: aws.String(desc),
			})
		}

		restApiId, _ := bodyData["restApiId"].(string)
		input := &apigateway.UpdateRestApiInput{
			RestApiId:       aws.String(restApiId),
			PatchOperations: patchOperations,
		}

		result, err := h.apigateway.UpdateRestApi(ctx, input)
		if err != nil {
			log.Printf("UpdateRestApi error: %v", err)
			sendError(c, http.StatusInternalServerError, "Failed to update REST API", err)
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	// AWS SDK format with patchOperations array
	log.Printf("UpdateRestApi: using AWS SDK patchOperations format")
	input := &apigateway.UpdateRestApiInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.UpdateRestApi(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update REST API", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getResources(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.GetResourcesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.GetResources(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get resources", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getResource(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.GetResourceInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.GetResource(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get resource", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createResource(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	log.Printf("createResource body: %s", string(bodyBytes))

	input := &apigateway.CreateResourceInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		log.Printf("createResource parse error: %v", err)
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	log.Printf("createResource input: RestApiId=%s, ParentId=%s, PathPart=%s",
		aws.ToString(input.RestApiId), aws.ToString(input.ParentId), aws.ToString(input.PathPart))

	result, err := h.apigateway.CreateResource(ctx, input)
	if err != nil {
		log.Printf("createResource error: %v", err)
		sendError(c, http.StatusInternalServerError, "Failed to create resource", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteResource(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.DeleteResourceInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.DeleteResource(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete resource", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) putMethod(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.PutMethodInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.PutMethod(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to put method", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getMethod(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.GetMethodInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.GetMethod(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get method", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteMethod(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.DeleteMethodInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.DeleteMethod(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete method", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) putIntegration(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.PutIntegrationInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	log.Printf("PutIntegration input: %+v", input)
	result, err := h.apigateway.PutIntegration(ctx, input)
	if err != nil {
		log.Printf("PutIntegration error: %v", err)
		sendError(c, http.StatusInternalServerError, "Failed to put integration", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getIntegration(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.GetIntegrationInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	log.Printf("GetIntegration input: %+v", input)
	result, err := h.apigateway.GetIntegration(ctx, input)
	if err != nil {
		log.Printf("GetIntegration error: %v", err)
		sendError(c, http.StatusInternalServerError, "Failed to get integration", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteIntegration(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.DeleteIntegrationInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.DeleteIntegration(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete integration", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createDeployment(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.CreateDeploymentInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.CreateDeployment(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create deployment", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteDeployment(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.DeleteDeploymentInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.DeleteDeployment(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete deployment", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createStage(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.CreateStageInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.CreateStage(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create stage", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getStages(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.GetStagesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.GetStages(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get stages", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) updateStage(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.UpdateStageInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.UpdateStage(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update stage", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteStage(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigateway.DeleteStageInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigateway.DeleteStage(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete stage", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// HTTP API v2 (ApiGatewayV2) handlers
func (h *ProxyHandler) getApis(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigatewayv2.GetApisInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigatewayv2.GetApis(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get HTTP APIs", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createApi(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigatewayv2.CreateApiInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigatewayv2.CreateApi(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create HTTP API", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteApi(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigatewayv2.DeleteApiInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigatewayv2.DeleteApi(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete HTTP API", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getApi(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigatewayv2.GetApiInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigatewayv2.GetApi(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get HTTP API", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getRoutes(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigatewayv2.GetRoutesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigatewayv2.GetRoutes(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get routes", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createRoute(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigatewayv2.CreateRouteInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigatewayv2.CreateRoute(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create route", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteRoute(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigatewayv2.DeleteRouteInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigatewayv2.DeleteRoute(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete route", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getIntegrationsV2(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigatewayv2.GetIntegrationsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigatewayv2.GetIntegrations(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get integrations", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createIntegrationV2(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &apigatewayv2.CreateIntegrationInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.apigatewayv2.CreateIntegration(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create integration", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== SSM HANDLERS ====================

func (h *ProxyHandler) handleSSM(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "GetParameter"):
		h.getParameter(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetParameters"):
		h.getParameters(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetParametersByPath"):
		h.getParametersByPath(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "PutParameter"):
		h.putParameter(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteParameter"):
		h.deleteParameter(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DescribeParameters"):
		h.describeParameters(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetParameterHistory"):
		h.getParameterHistory(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListTagsForResource"):
		h.listTagsForResource(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "AddTagsToResource"):
		h.addTagsToResource(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "RemoveTagsFromResource"):
		h.removeTagsFromResource(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown SSM action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) getParameter(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.GetParameterInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.GetParameter(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get parameter", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getParameters(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.GetParametersInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.GetParameters(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get parameters", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getParametersByPath(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.GetParametersByPathInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.GetParametersByPath(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get parameters by path", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) putParameter(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.PutParameterInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.PutParameter(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to put parameter", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteParameter(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.DeleteParameterInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.DeleteParameter(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete parameter", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) describeParameters(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.DescribeParametersInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.DescribeParameters(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to describe parameters", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getParameterHistory(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.GetParameterHistoryInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.GetParameterHistory(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get parameter history", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listTagsForResource(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.ListTagsForResourceInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.ListTagsForResource(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list tags for resource", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) addTagsToResource(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.AddTagsToResourceInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.AddTagsToResource(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to add tags to resource", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) removeTagsFromResource(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &ssm.RemoveTagsFromResourceInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.ssm.RemoveTagsFromResource(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to remove tags from resource", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== IAM HANDLERS ====================

func (h *ProxyHandler) handleIAM(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "CreateUser"):
		h.createUser(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetUser"):
		h.getUser(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListUsers"):
		h.listUsers(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteUser"):
		h.deleteUser(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateRole"):
		h.createRole(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetRole"):
		h.getRole(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListRoles"):
		h.listRoles(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteRole"):
		h.deleteRole(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListPolicies"):
		h.listPolicies(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetPolicy"):
		h.getPolicy(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateAccessKey"):
		h.createAccessKey(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListAccessKeys"):
		h.listAccessKeys(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteAccessKey"):
		h.deleteAccessKey(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateAccessKeyStatus"):
		h.updateAccessKeyStatus(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "AttachRolePolicy"):
		h.attachRolePolicy(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DetachRolePolicy"):
		h.detachRolePolicy(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListAttachedRolePolicies"):
		h.listAttachedRolePolicies(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateGroup"):
		h.createGroup(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetGroup"):
		h.getGroup(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListGroups"):
		h.listGroups(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteGroup"):
		h.deleteGroup(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "AddUserToGroup"):
		h.addUserToGroup(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "RemoveUserFromGroup"):
		h.removeUserFromGroup(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListGroupsForUser"):
		h.listGroupsForUser(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListUserPolicies"):
		h.listUserPolicies(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListRolePolicies"):
		h.listRolePolicies(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetRolePolicy"):
		h.getRolePolicy(ctx, c, bodyBytes)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown IAM action: " + xAmzTarget})
	}
}

func (h *ProxyHandler) createUser(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.CreateUserInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.CreateUser(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create user", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getUser(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.GetUserInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.GetUser(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get user", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listUsers(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.ListUsersInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.ListUsers(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list users", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteUser(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.DeleteUserInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.DeleteUser(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete user", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createRole(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.CreateRoleInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.CreateRole(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create role", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getRole(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.GetRoleInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.GetRole(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get role", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listRoles(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.ListRolesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.ListRoles(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list roles", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteRole(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.DeleteRoleInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.DeleteRole(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete role", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listPolicies(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.ListPoliciesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.ListPolicies(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list policies", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getPolicy(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.GetPolicyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.GetPolicy(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get policy", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createAccessKey(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.CreateAccessKeyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.CreateAccessKey(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create access key", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listAccessKeys(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.ListAccessKeysInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.ListAccessKeys(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list access keys", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteAccessKey(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.DeleteAccessKeyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.DeleteAccessKey(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete access key", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) updateAccessKeyStatus(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.UpdateAccessKeyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.UpdateAccessKey(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update access key status", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) attachRolePolicy(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.AttachRolePolicyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.AttachRolePolicy(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to attach role policy", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) detachRolePolicy(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.DetachRolePolicyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.DetachRolePolicy(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to detach role policy", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listAttachedRolePolicies(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.ListAttachedRolePoliciesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.ListAttachedRolePolicies(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list attached role policies", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createGroup(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.CreateGroupInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.CreateGroup(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create group", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getGroup(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.GetGroupInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.GetGroup(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get group", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listGroups(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.ListGroupsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.ListGroups(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list groups", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteGroup(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.DeleteGroupInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.DeleteGroup(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete group", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) addUserToGroup(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.AddUserToGroupInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.AddUserToGroup(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to add user to group", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) removeUserFromGroup(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.RemoveUserFromGroupInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.RemoveUserFromGroup(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to remove user from group", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listGroupsForUser(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.ListGroupsForUserInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.ListGroupsForUser(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list groups for user", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listUserPolicies(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.ListUserPoliciesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.ListUserPolicies(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list user policies", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listRolePolicies(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.ListRolePoliciesInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.ListRolePolicies(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list role policies", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getRolePolicy(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &iam.GetRolePolicyInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.iam.GetRolePolicy(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get role policy", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== KINESIS HANDLERS ====================

func (h *ProxyHandler) handleKinesis(c *gin.Context) {
	xAmzTarget := c.GetHeader("X-Amz-Target")
	bodyBytes := readBody(c)
	ctx := context.Background()

	switch {
	case strings.Contains(xAmzTarget, "ListStreams"):
		h.listStreams(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "CreateStream"):
		h.createStream(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DeleteStream"):
		h.deleteStream(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DescribeStream"):
		h.describeStream(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DescribeStreamSummary"):
		h.describeStreamSummary(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "ListShards"):
		h.listShards(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetShardIterator"):
		h.getShardIterator(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "GetRecords"):
		h.getRecords(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "PutRecord"):
		h.putRecord(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "PutRecords"):
		h.putRecords(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "MergeShards"):
		h.mergeShards(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "SplitShard"):
		h.splitShard(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "UpdateShardCount"):
		h.updateShardCount(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "EnableEnhancedMonitoring"):
		h.enableEnhancedMonitoring(ctx, c, bodyBytes)
	case strings.Contains(xAmzTarget, "DisableEnhancedMonitoring"):
		h.disableEnhancedMonitoring(ctx, c, bodyBytes)
	default:
		sendError(c, http.StatusNotFound, "Kinesis operation not supported: "+xAmzTarget, nil)
	}
}

func (h *ProxyHandler) listStreams(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.ListStreamsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.ListStreams(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list streams", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) createStream(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.CreateStreamInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.CreateStream(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create stream", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) deleteStream(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.DeleteStreamInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.DeleteStream(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete stream", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) describeStream(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.DescribeStreamInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.DescribeStream(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to describe stream", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) describeStreamSummary(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.DescribeStreamSummaryInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.DescribeStreamSummary(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to describe stream summary", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) listShards(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.ListShardsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.ListShards(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list shards", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getShardIterator(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.GetShardIteratorInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.GetShardIterator(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get shard iterator", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) getRecords(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.GetRecordsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.GetRecords(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get records", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) putRecord(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.PutRecordInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.PutRecord(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to put record", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) putRecords(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.PutRecordsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.PutRecords(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to put records", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) mergeShards(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.MergeShardsInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.MergeShards(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to merge shards", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) splitShard(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.SplitShardInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.SplitShard(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to split shard", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) updateShardCount(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.UpdateShardCountInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.UpdateShardCount(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update shard count", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) enableEnhancedMonitoring(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.EnableEnhancedMonitoringInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.EnableEnhancedMonitoring(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to enable enhanced monitoring", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *ProxyHandler) disableEnhancedMonitoring(ctx context.Context, c *gin.Context, bodyBytes []byte) {
	input := &kinesis.DisableEnhancedMonitoringInput{}
	if err := parseBody(c, bodyBytes, input); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	result, err := h.kinesis.DisableEnhancedMonitoring(ctx, input)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to disable enhanced monitoring", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ==================== UTILITIES ====================

func readBody(c *gin.Context) []byte {
	if c.Request.Body != nil {
		if bodyBytes, err := io.ReadAll(c.Request.Body); err == nil {
			return bodyBytes
		}
	}
	return nil
}

// parseBody parses JSON body into the given pointer with proper error handling.
// Returns an error response if parsing fails.
func parseBody(c *gin.Context, bodyBytes []byte, target interface{}) error {
	if len(bodyBytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(bodyBytes, target); err != nil {
		return fmt.Errorf("failed to parse request body: %w", err)
	}
	return nil
}

// sendError sends a JSON error response and logs the error
func sendError(c *gin.Context, status int, message string, err error) {
	if err != nil {
		log.Printf("%s: %v", message, err)
	}
	c.JSON(status, gin.H{"error": message})
}

func (h *ProxyHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":       "healthy",
		"proxy":        "aws-proxy",
		"target":       h.cfg.AwsEndpoint,
		"endpoint_url": h.cfg.AwsEndpoint,
	})
}

func (h *ProxyHandler) BackendHealthCheck(c *gin.Context) {
	testURLs := []string{
		h.cfg.AwsEndpoint + "/",
		h.cfg.AwsEndpoint + "/_health",
		h.cfg.AwsEndpoint + "/health",
	}

	for _, targetURL := range testURLs {
		req, _ := http.NewRequest("GET", targetURL, nil)
		client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				c.JSON(http.StatusOK, gin.H{
					"status":     "healthy",
					"backend":    "reachable",
					"target":     h.cfg.AwsEndpoint,
					"statusCode": resp.StatusCode,
				})
				return
			}
		}
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{
		"status":  "unhealthy",
		"backend": "unreachable",
		"target":  h.cfg.AwsEndpoint,
	})
}

func setupRoutes(r *gin.Engine, handler *ProxyHandler) {
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH, HEAD")
		c.Header("Access-Control-Allow-Headers",
			"Content-Type, Authorization, X-Requested-With, "+
				"X-Amz-Date, X-Amz-Security-Token, X-Api-Key, "+
				"x-amz-content-sha256, x-amz-target, x-amz-user-agent, "+
				"x-amz-id-2, x-amz-request-id, Accept, Accept-Encoding, "+
				"Content-Length, Host, User-Agent, "+
				"x-amz-invocation-type, x-amz-log-type, x-amz-client-context, "+
				"amz-sdk-request, amz-sdk-invocation-id, amz-content-sha256")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	r.GET("/health", handler.BackendHealthCheck)
	r.GET("/_health", handler.BackendHealthCheck)
	r.GET("/proxy/health", handler.HealthCheck)

	r.Any("/:service/*path", handler.serviceRouter)
}

func main() {
	cfg := loadConfig()

	log.Printf("Starting AWS Proxy Server...")
	log.Printf("  Port: %s", cfg.Port)
	log.Printf("  AWS Endpoint: %s", cfg.AwsEndpoint)
	log.Printf("  AWS Region: %s", cfg.AwsRegion)

	handler := NewProxyHandler(cfg)

	r := gin.Default()
	r.Use(gin.Logger())

	setupRoutes(r, handler)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server listening on %s", addr)
	log.Printf("Proxy endpoints:")
	log.Printf("  Secrets Manager: http://localhost:%s/secretsmanager/", cfg.Port)
	log.Printf("  S3:              http://localhost:%s/s3/", cfg.Port)
	log.Printf("  Lambda:          http://localhost:%s/lambda/", cfg.Port)
	log.Printf("  SQS:             http://localhost:%s/sqs/", cfg.Port)
	log.Printf("  SNS:             http://localhost:%s/sns/", cfg.Port)
	log.Printf("  KMS:             http://localhost:%s/kms/", cfg.Port)
	log.Printf("  DynamoDB:        http://localhost:%s/dynamodb/", cfg.Port)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
