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
curl http://localhost:8081/proxy/health

# List S3 buckets (requires LocalStack running)
curl -X GET http://localhost:8081/s3/ \
  -H "X-Amz-Target: ListBuckets"
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
export AWS_ENDPOINT=http://localhost:4566  # AWS backend
export AWS_REGION=us-east-1           # AWS region
export AWS_ACCESS_KEY=test            # Access key
export AWS_SECRET_KEY=test             # Secret key
```

### Configuration File

Create `config.yaml` or `config.json`:

```yaml
port: "8081"
aws_endpoint: "http://localhost:4566"
aws_region: "us-east-1"
aws_access_key: "test"
aws_secret_key: "test"
```

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
│   └── wire.go            # Dependency injection
└── internal/
    ├── config/             # Configuration (ayotl)
    ├── ports/             # Interface definitions
    ├── application/       # Business logic
    └── adapters/
        ├── aws/           # AWS SDK adapters
        └── http/          # HTTP handlers
```

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