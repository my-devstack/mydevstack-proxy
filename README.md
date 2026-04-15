# mydevstack-proxy

A lightweight AWS proxy service built in Go that provides a unified HTTP interface for multiple AWS services. Perfect for local development with LocalStack or testing environments.

## Description

mydevstack-proxy acts as a gateway between your application and AWS services. Instead of making direct AWS SDK calls, your application sends HTTP requests to this proxy, which routes them to the appropriate AWS service backend.

This is particularly useful for:
- **Local Development** - Work with AWS services locally using LocalStack
- **Testing** - Simplified integration testing without real AWS credentials
- **Debugging** - Easy to inspect and log AWS API requests
- **Docker Environments** - Lightweight alternative to full AWS SDK setup

## Quick Start

### Prerequisites

- **Go 1.21+** - Download from [go.dev](https://go.dev)
- **LocalStack** (optional) - For local AWS services

### Run the Server

```bash
# Clone and enter directory
git clone https://github.com/my-devstack/mydevstack-proxy.git
cd mydevstack-proxy

# Run with default settings (LocalStack on port 4566)
go run main.go

# Or specify custom port
PORT=8081 go run main.go
```

### Make Your First Request

```bash
# Test health endpoint
curl http://localhost:8081/_health

# List S3 buckets (requires LocalStack running)
curl http://localhost:8081/s3/ -H "X-Amz-Target: ListBuckets"
```

### Change Region Dynamically

```bash
# Switch to a different region (e.g., eu-west-2)
curl -X POST http://localhost:8081/proxy/region \
  -H "Content-Type: application/json" \
  -d '{"region": "eu-west-2"}'
```

## Requirements

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.21+ | Required |
| LocalStack | Latest | Optional, for local AWS |
| Git | Any | For cloning |

## Configuration

### Environment Variables

```bash
export PORT=8081                      # Server port (default: 8081)
export AWS_ENDPOINT=http://localhost:4566  # AWS backend (LocalStack)
export AWS_ACCESS_KEY=test            # Access key
export AWS_SECRET_KEY=test             # Secret key
```

### Configuration File

Create `config.yaml` or `config.json`:

```yaml
port: "8081"
aws_endpoint: "http://localhost:4566"
aws_access_key: "test"
aws_secret_key: "test"
service_pattern: "root"
```

**Note:** Region is set dynamically via the `/proxy/region` endpoint, not in configuration. Default region is `us-east-1`.

## Supported AWS Services

| Service | Endpoint | Example |
|---------|----------|---------|
| S3 | `/s3/*` | `curl /s3/ -H "X-Amz-Target: ListBuckets"` |
| Lambda | `/lambda/*` | `curl /lambda/ -H "X-Amz-Target: Invoke"` |
| Secrets Manager | `/secretsmanager/*` | `curl /secretsmanager/ -H "X-Amz-Target: GetSecretValue"` |
| SQS | `/sqs/*` | `curl /sqs/ -H "X-Amz-Target: ListQueues"` |
| SNS | `/sns/*` | `curl /sns/ -H "X-Amz-Target: ListTopics"` |
| KMS | `/kms/*` | `curl /kms/ -H "X-Amz-Target: ListKeys"` |
| DynamoDB | `/dynamodb/*` | `curl /dynamodb/ -H "X-Amz-Target: ListTables"` |
| API Gateway | `/apigateway/*` | `curl /apigateway/ -H "X-Amz-Target: GetRestApis"` |
| API Gateway V2 | `/apigatewayv2/*` | `curl /apigatewayv2/ -H "X-Amz-Target: GetApis"` |
| SSM | `/ssm/*` | `curl /ssm/ -H "X-Amz-Target: GetParameter"` |
| IAM | `/iam/*` | `curl /iam/ -H "X-Amz-Target: ListUsers"` |
| Kinesis | `/kinesis/*` | `curl /kinesis/ -H "X-Amz-Target: ListStreams"` |
| RDS | `/rds/*` | `curl /rds/ -H "X-Amz-Target: DescribeDBInstances"` |
| ElastiCache | `/elasticache/*` | `curl /elasticache/ -H "X-Amz-Target: DescribeCacheClusters"` |

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Your Application                     │
└─────────────────────┬───────────────────────────────────┘
                       │ HTTP Request + X-Amz-Target
                       ▼
┌─────────────────────────────────────────────────────────┐
│                   mydevstack-proxy                       │
│  ┌─────────────┐    ┌─────────────┐    ┌────────────┐ │
│  │   Handler   │ →  │ Application │ →  │   AWS      │ │
│  │   (HTTP)    │    │   Service   │    │  Adapters  │ │
│  └─────────────┘    └─────────────┘    └────────────┘ │
│                           │                              │
│                    SetRegion()                          │
│                    SetServices() ◄─ Recreates adapters │
└─────────────────────┬───────────────────────────────────┘
                       │ Forward to AWS Endpoint
                       ▼
┌─────────────────────────────────────────────────────────┐
│              LocalStack / Real AWS                       │
└─────────────────────────────────────────────────────────┘
```

### Project Structure

```
.
├── main.go                 # Application entry point
├── bootstrap/
│   └── wire.go            # Dependency injection container
└── internal/
    ├── config/             # Configuration loader (ayotl)
    ├── ports/             # Interface definitions (interfaces)
    ├── application/       # Business logic (ProxyService)
    │   └── service.go     # SetRegion(), SetServices()
    └── adapters/
        ├── aws/           # AWS SDK adapters (12 services)
        └── http/          # HTTP handlers
```

### Key Design Patterns

1. **Lazy Initialization** - AWS clients are created in `SetServices()`, not in constructor
2. **Region Switching** - Call `POST /proxy/region` to change region and recreate all adapters
3. **Interface-Driven** - Ports define contracts, Adapters implement them

## X-Amz-Target Header

The `X-Amz-Target` header is a **standard AWS header** that specifies which AWS API operation to call. This proxy uses this header to:

1. **Route** the request to the correct AWS service handler
2. **Forward** the request to the configured AWS endpoint

Examples:
- `X-Amz-Target: ListBuckets` → S3
- `X-Amz-Target: Invoke` → Lambda
- `X-Amz-Target: GetSecretValue` → Secrets Manager

## Development

```bash
# Run tests
go test -v ./...

# Run with hot reload (using air)
go install github.com/cosmtrek/air@latest
air

# Lint code
golangci-lint run
```

### Using Mockery for Interface Mocks

This project uses [mockery](https://github.com/vektra/mockery) to generate mocks for interfaces in unit tests.

#### Generating Mocks

To generate all mocks defined in `.mockery.yaml`:

```bash
# Install mockery (if not already installed)
go install github.com/vektra/mockery/v2@latest

# Generate all mocks
mockery
```

#### Configuration

The `.mockery.yaml` file is located in the project root and configures how mocks are generated:

```yaml
all: true                           # Generate mocks for all interfaces in the package
structname: '{{.InterfaceName}}'   # Use interface name as struct name
template: testify                   # Use testify template for mock generation
packages:
  github.com/my-devstack/mydevstack-proxy/internal/ports:
    config:
      dir: mocks/ports              # Output directory for mocks
      filename: "{{.InterfaceName}}.go"  # File naming pattern
```

#### Adding New Interfaces

To add mocks for a new interface:

1. Ensure the interface is defined in `internal/ports/`
2. Run `mockery` to regenerate all mocks (the new interface will be auto-detected)
3. Use the generated mock in tests:

```go
import mockports "github.com/my-devstack/mydevstack-proxy/mocks/ports"

// In your test file:
mockS3 := mockports.NewS3Port(t)
mockS3.EXPECT().ListBuckets(mock.Anything).Return(&s3.ListBucketsOutput{}, nil)
```

#### Mock Location

Generated mocks are stored in `mocks/ports/` directory:

- `ProxyService.go` - Mock for ProxyService interface
- `S3Port.go` - Mock for S3Port interface
- `LambdaPort.go` - Mock for LambdaPort interface
- `SecretsManagerPort.go` - Mock for SecretsManagerPort interface
- `SQSPort.go` - Mock for SQSPort interface
- `SNSPort.go` - Mock for SNSPort interface
- `KMSPort.go` - Mock for KMSPort interface
- `DynamoDBPort.go` - Mock for DynamoDBPort interface
- `APIGatewayPort.go` - Mock for APIGatewayPort interface
- `APIGatewayV2Port.go` - Mock for APIGatewayV2Port interface
- `SSMPort.go` - Mock for SSMPort interface
- `IAMPort.go` - Mock for IAMPort interface
- `KinesisPort.go` - Mock for KinesisPort interface
- `RDSPort.go` - Mock for RDSPort interface
- `ElastiCachePort.go` - Mock for ElastiCachePort interface

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on:

- Development setup
- Code standards
- Submitting changes

## Code of Conduct

Please read our [Code of Conduct](CODE_OF_CONDUCT.md) to keep our community approachable and respectful.

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE) for details.

---

Built with Go and AWS SDK v2