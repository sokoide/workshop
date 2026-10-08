package minio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// newTestAdapter は httptest サーバーを S3 エンドポイントに見立てた
// Adapter を構築する。実 MinIO コンテナを必要としない。
func newTestAdapter(t *testing.T, handler http.Handler) (*Adapter, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
		config.WithBaseEndpoint(server.URL),
	)
	if err != nil {
		t.Fatalf("failed to load SDK config: %v", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // main.go と同じ path-style 設定
	})
	return NewAdapter(client, "workshop-images"), server
}

func TestAdapter_Upload(t *testing.T) {
	type request struct {
		method string
		path   string
	}
	requests := make(chan request, 4)

	adapter, _ := newTestAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- request{method: r.Method, path: r.URL.Path}
		w.WriteHeader(http.StatusOK)
	}))

	if err := adapter.Upload(context.Background(), "photo.png", []byte("png bytes")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case req := <-requests:
		// path-style なのでバケット名が URL パスに乗る。
		if req.method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.method)
		}
		if req.path != "/workshop-images/photo.png" {
			t.Errorf("expected /workshop-images/photo.png, got %s", req.path)
		}
	default:
		t.Fatal("no request reached the server")
	}
}

func TestAdapter_UploadWrapsBackendErrors(t *testing.T) {
	adapter, _ := newTestAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))

	err := adapter.Upload(context.Background(), "photo.png", []byte("png bytes"))
	if err == nil {
		t.Fatal("expected an error on a 500 response")
	}
	if !strings.Contains(err.Error(), "failed to upload object to minio") {
		t.Errorf("error should be wrapped by the adapter, got %v", err)
	}
}

func TestAdapter_GeneratePresignedURL(t *testing.T) {
	// 署名はローカルで行われるため、ハンドラが呼ばれることはない。
	adapter, _ := newTestAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("presigning must not issue a request, got %s %s", r.Method, r.URL.Path)
	}))

	raw, err := adapter.GeneratePresignedURL(context.Background(), "photo.png", 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("generated URL is not parseable: %v", err)
	}
	if !strings.HasSuffix(parsed.Path, "/workshop-images/photo.png") {
		t.Errorf("URL path should target the object, got %s", parsed.Path)
	}
	if got := parsed.Query().Get("X-Amz-Expires"); got != "900" {
		t.Errorf("expected X-Amz-Expires=900 (15m), got %q", got)
	}
	// SigV4 署名が付いていることも確認する。
	for _, param := range []string{"X-Amz-Signature", "X-Amz-Credential", "X-Amz-Algorithm"} {
		if parsed.Query().Get(param) == "" {
			t.Errorf("presigned URL is missing %s", param)
		}
	}
}
