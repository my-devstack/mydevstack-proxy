package application

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsadapter "github.com/my-devstack/mydevstack-proxy/internal/adapters/aws"
	configloader "github.com/my-devstack/mydevstack-proxy/internal/config"
	"github.com/my-devstack/mydevstack-proxy/internal/ports"
)

type ProxyService struct {
	cfg            *configloader.Config
	secretsManager ports.SecretsManagerPort
	s3             ports.S3Port
	lambda         ports.LambdaPort
	sqs            ports.SQSPort
	sns            ports.SNSPort
	kms            ports.KMSPort
	dynamodb       ports.DynamoDBPort
	apigateway     ports.APIGatewayPort
	apigatewayv2   ports.APIGatewayV2Port
	ssm            ports.SSMPort
	iam            ports.IAMPort
	kinesis        ports.KinesisPort
}

func NewProxyService(cfg *configloader.Config) (ports.ProxyService, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.AwsRegion),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AwsAccessKey,
			cfg.AwsSecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	return &ProxyService{
		cfg:            cfg,
		secretsManager: awsadapter.NewSecretsManagerAdapter(awsCfg, cfg.AwsEndpoint),
		s3:             awsadapter.NewS3Adapter(awsCfg, cfg.AwsEndpoint),
		lambda:         awsadapter.NewLambdaAdapter(awsCfg, cfg.AwsEndpoint),
		sqs:            awsadapter.NewSQSAdapter(awsCfg, cfg.AwsEndpoint),
		sns:            awsadapter.NewSNSAdapter(awsCfg, cfg.AwsEndpoint),
		kms:            awsadapter.NewKMSAdapter(awsCfg, cfg.AwsEndpoint),
		dynamodb:       awsadapter.NewDynamoDBAdapter(awsCfg, cfg.AwsEndpoint),
		apigateway:     awsadapter.NewAPIGatewayAdapter(awsCfg, cfg.AwsEndpoint),
		apigatewayv2:   awsadapter.NewAPIGatewayV2Adapter(awsCfg, cfg.AwsEndpoint),
		ssm:            awsadapter.NewSSMAdapter(awsCfg, cfg.AwsEndpoint),
		iam:            awsadapter.NewIAMAdapter(awsCfg, cfg.AwsEndpoint),
		kinesis:        awsadapter.NewKinesisAdapter(awsCfg, cfg.AwsEndpoint),
	}, nil
}

func (s *ProxyService) Config() *configloader.Config {
	return s.cfg
}

func (s *ProxyService) SecretsManager() ports.SecretsManagerPort {
	return s.secretsManager
}

func (s *ProxyService) S3() ports.S3Port {
	return s.s3
}

func (s *ProxyService) Lambda() ports.LambdaPort {
	return s.lambda
}

func (s *ProxyService) SQS() ports.SQSPort {
	return s.sqs
}

func (s *ProxyService) SNS() ports.SNSPort {
	return s.sns
}

func (s *ProxyService) KMS() ports.KMSPort {
	return s.kms
}

func (s *ProxyService) DynamoDB() ports.DynamoDBPort {
	return s.dynamodb
}

func (s *ProxyService) APIGateway() ports.APIGatewayPort {
	return s.apigateway
}

func (s *ProxyService) APIGatewayV2() ports.APIGatewayV2Port {
	return s.apigatewayv2
}

func (s *ProxyService) SSM() ports.SSMPort {
	return s.ssm
}

func (s *ProxyService) IAM() ports.IAMPort {
	return s.iam
}

func (s *ProxyService) Kinesis() ports.KinesisPort {
	return s.kinesis
}

var _ ports.ProxyService = (*ProxyService)(nil)
