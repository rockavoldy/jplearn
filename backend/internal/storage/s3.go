// Package storage wraps an S3-compatible object store (AWS S3, Cloudflare R2, MinIO).
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"jplearn/internal/config"
)

// S3Client provides object operations against a single bucket.
type S3Client struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
}

// New builds an S3 client from configuration. It works with any S3-compatible
// endpoint when S3_ENDPOINT is set (MinIO/R2) or with AWS when it is empty.
func New(ctx context.Context, cfg *config.Config) (*S3Client, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.S3Region),
	}
	if cfg.S3AccessKeyID != "" && cfg.S3SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
		o.UsePathStyle = cfg.S3UsePathStyle
	})

	return &S3Client{
		client:        client,
		bucket:        cfg.S3Bucket,
		publicBaseURL: cfg.S3PublicBaseURL,
	}, nil
}

// PutStream uploads the given reader to key with the provided content type.
// Returns the object key and a resolvable URL.
func (s *S3Client) PutStream(ctx context.Context, key, contentType string, body io.Reader) (string, string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", "", fmt.Errorf("put object %s: %w", key, err)
	}
	return key, s.URLFor(key), nil
}

// PutBytes is a convenience wrapper around PutStream for in-memory payloads.
func (s *S3Client) PutBytes(ctx context.Context, key, contentType string, data []byte) (string, string, error) {
	return s.PutStream(ctx, key, contentType, bytes.NewReader(data))
}

// Get streams an object from the bucket.
func (s *S3Client) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("get object %s: %w", key, err)
	}
	ct := "application/octet-stream"
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return out.Body, ct, nil
}

// URLFor returns a URL for the given key. When a public base URL is configured
// it is used; otherwise the API image proxy path is returned.
func (s *S3Client) URLFor(key string) string {
	if s.publicBaseURL != "" {
		return fmt.Sprintf("%s/%s", trimTrailingSlash(s.publicBaseURL), key)
	}
	return key
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
