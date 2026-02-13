package minio

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Adapter struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

func NewAdapter(client *s3.Client, bucket string) *Adapter {
	return &Adapter{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        bucket,
	}
}

func (a *Adapter) Upload(ctx context.Context, key string, data []byte) error {
	_, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload object to minio: %w", err)
	}
	return nil
}

func (a *Adapter) GeneratePresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	req, err := a.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL from minio: %w", err)
	}
	return req.URL, nil
}
