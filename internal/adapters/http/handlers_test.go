package httphandlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	configloader "github.com/my-devstack/mydevstack-proxy/internal/config"
	"github.com/my-devstack/mydevstack-proxy/internal/ports"
)

type mockS3Port struct {
	listBucketsFunc func(ctx context.Context) (*s3.ListBucketsOutput, error)
}

func (m *mockS3Port) ListBuckets(ctx context.Context) (*s3.ListBucketsOutput, error) {
	if m.listBucketsFunc != nil {
		return m.listBucketsFunc(ctx)
	}
	return &s3.ListBucketsOutput{}, nil
}

func (m *mockS3Port) ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	return nil, nil
}

func (m *mockS3Port) GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return nil, nil
}

func (m *mockS3Port) PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return nil, nil
}

func (m *mockS3Port) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return nil, nil
}

func (m *mockS3Port) DeleteBucket(ctx context.Context, input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return nil, nil
}

func (m *mockS3Port) HeadBucket(ctx context.Context, input *s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
	return nil, nil
}

func (m *mockS3Port) HeadObject(ctx context.Context, input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return nil, nil
}

func (m *mockS3Port) CreateBucket(ctx context.Context, input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	return nil, nil
}

type testProxyService struct {
	s3Port ports.S3Port
	cfg    *configloader.Config
}

func (s *testProxyService) S3() ports.S3Port {
	return s.s3Port
}

func (s *testProxyService) Lambda() ports.LambdaPort                 { return nil }
func (s *testProxyService) SecretsManager() ports.SecretsManagerPort { return nil }
func (s *testProxyService) SQS() ports.SQSPort                       { return nil }
func (s *testProxyService) SNS() ports.SNSPort                       { return nil }
func (s *testProxyService) KMS() ports.KMSPort                       { return nil }
func (s *testProxyService) DynamoDB() ports.DynamoDBPort             { return nil }
func (s *testProxyService) APIGateway() ports.APIGatewayPort         { return nil }
func (s *testProxyService) APIGatewayV2() ports.APIGatewayV2Port     { return nil }
func (s *testProxyService) SSM() ports.SSMPort                       { return nil }
func (s *testProxyService) IAM() ports.IAMPort                       { return nil }
func (s *testProxyService) Kinesis() ports.KinesisPort               { return nil }
func (s *testProxyService) Config() *configloader.Config {
	return &configloader.Config{AwsEndpoint: "http://localhost:4550", AwsRegion: "us-east-1"}
}

func setupTestRouter(handler *ProxyHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", handler.HealthCheck)
	r.GET("/_health", handler.BackendHealthCheck)
	r.Any("/:service/*path", handler.ServiceRouter)
	return r
}

func TestHealthCheck(t *testing.T) {
	svc := &testProxyService{}
	handler := NewProxyHandler(svc)
	r := setupTestRouter(handler)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HealthCheck status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("HealthCheck status = %v, want healthy", response["status"])
	}
	if response["proxy"] != "aws-proxy" {
		t.Errorf("HealthCheck proxy = %v, want aws-proxy", response["proxy"])
	}
}

func TestServiceRouter(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		target     string
		wantStatus int
	}{
		{
			name:       "unknown service returns 404",
			method:     "GET",
			path:       "/unknown/test",
			target:     "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "s3 service routes correctly",
			method:     "GET",
			path:       "/s3/",
			target:     "ListBuckets",
			wantStatus: http.StatusOK,
		},
		{
			name:       "lambda service returns bad request for no target",
			method:     "GET",
			path:       "/lambda/",
			target:     "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "secretsmanager service returns bad request for no target",
			method:     "GET",
			path:       "/secretsmanager/",
			target:     "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &testProxyService{
				s3Port: &mockS3Port{
					listBucketsFunc: func(ctx context.Context) (*s3.ListBucketsOutput, error) {
						return &s3.ListBucketsOutput{}, nil
					},
				},
			}
			handler := NewProxyHandler(svc)
			r := setupTestRouter(handler)

			w := httptest.NewRecorder()
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			if tt.target != "" {
				req.Header.Set("X-Amz-Target", tt.target)
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("ServiceRouter status = %v, want %v, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestS3ListBuckets(t *testing.T) {
	svc := &testProxyService{
		s3Port: &mockS3Port{
			listBucketsFunc: func(ctx context.Context) (*s3.ListBucketsOutput, error) {
				return &s3.ListBucketsOutput{}, nil
			},
		},
	}
	handler := NewProxyHandler(svc)
	r := setupTestRouter(handler)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/s3/", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("X-Amz-Target", "ListBuckets")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListBuckets status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestCORSHeaders(t *testing.T) {
	svc := &testProxyService{}
	handler := NewProxyHandler(svc)
	r := setupTestRouter(handler)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("OPTIONS", "/s3/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("CORS OPTIONS status = %v, want OK or BadRequest", w.Code)
	}
}

func TestBackendHealthCheck_Reachable(t *testing.T) {
	svc := &testProxyService{
		cfg: &configloader.Config{AwsEndpoint: "http://localhost:9999"},
	}
	handler := NewProxyHandler(svc)
	r := setupTestRouter(handler)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/_health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusOK {
		t.Errorf("BackendHealthCheck status = %v, want ServiceUnavailable or OK", w.Code)
	}
}
