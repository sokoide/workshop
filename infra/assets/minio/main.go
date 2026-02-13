package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/sokoide/workshop/infra/assets/minio/internal/infra/minio"
	"github.com/sokoide/workshop/infra/assets/minio/internal/usecase"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go [upload|share] [filename]")
		os.Exit(1)
	}

	cmd := os.Args[1]
	filename := os.Args[2]

	ctx := context.Background()
	const (
		endpointURL = "http://localhost:9000"
		region      = "us-east-1"
		bucketName  = "workshop-images"
		accessKey   = "minioadmin"
		secretKey   = "minioadmin"
	)

	// AWS SDK Config for MinIO
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               endpointURL,
					HostnameImmutable: true,
				}, nil
			})),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg)

	// Ensure bucket exists
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		var bnea *types.BucketAlreadyExists
		var bnaie *types.BucketAlreadyOwnedByYou
		if !errors.As(err, &bnea) && !errors.As(err, &bnaie) {
			log.Printf("Warning: failed to create bucket (may already exist): %v", err)
		}
	}

	minioAdapter := minio.NewAdapter(client, bucketName)
	fileUsecase := usecase.NewFileUsecase(minioAdapter)

	switch cmd {
	case "upload":
		if err := fileUsecase.UploadFile(ctx, filename); err != nil {
			log.Fatalf("failed to upload: %v", err)
		}
		fmt.Printf("Successfully uploaded %s to bucket %s\n", filename, bucketName)

	case "share":
		url, err := fileUsecase.GetShareLink(ctx, filename)
		if err != nil {
			log.Fatalf("failed to get share link: %v", err)
		}
		fmt.Printf("Presigned URL (valid for 15m):\n%s\n", url)

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
	}
}
