package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/retry"
	"github.com/karavanix/karavantrack-api-server/pkg/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type S3Client struct {
	client      *minio.Client
	retryConfig retry.RetryConfig
}

func New(opts ...Option) (*S3Client, error) {
	defaultOptions := &Options{
		Endpoint:  "localhost:9000",
		Region:    "us-east-1",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Secure:    false,
	}

	for _, opt := range opts {
		opt(defaultOptions)
	}

	creds := credentials.NewStaticV4(defaultOptions.AccessKey, defaultOptions.SecretKey, "")

	minioOpts := &minio.Options{
		Creds:        creds,
		Region:       defaultOptions.Region,
		Secure:       defaultOptions.Secure,
		BucketLookup: minio.BucketLookupAuto,
		Transport:    otelhttp.NewTransport(utils.DefaultInsecureTransport()),
	}

	client, err := minio.New(defaultOptions.Endpoint, minioOpts)
	if err != nil {
		return nil, err
	}
	return &S3Client{
		client:      client,
		retryConfig: retry.DefaultConfig(),
	}, nil
}

func (c *S3Client) EndpointURL() string {
	return c.client.EndpointURL().String()
}

// EnsureBucket makes sure bucket exists; safe under races.
func (c *S3Client) EnsureBucket(ctx context.Context, bucket string) error {
	_, err := retry.Retry(ctx, c.retryConfig, func(ctx context.Context) (struct{}, error) {
		exists, err := c.client.BucketExists(ctx, bucket)
		if err != nil {
			return struct{}{}, fmt.Errorf("s3: bucket exists check %q: %w", bucket, err)
		}
		if exists {
			return struct{}{}, nil
		}

		// MakeBucket can fail if created concurrently. Re-check on failure.
		if err := c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			exists2, err2 := c.client.BucketExists(ctx, bucket)
			if err2 == nil && exists2 {
				return struct{}{}, nil
			}
			return struct{}{}, fmt.Errorf("s3: create bucket %q: %w", bucket, err)
		}
		return struct{}{}, nil
	})
	return err
}

type PutOptions struct {
	ContentType string
	Metadata    map[string]string
}

func (c *S3Client) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, opt PutOptions) (etag string, err error) {
	if bucket == "" {
		return "", fmt.Errorf("s3: bucket is required")
	}
	if key == "" {
		return "", fmt.Errorf("s3: key is required")
	}
	if r == nil {
		return "", fmt.Errorf("s3: nil reader")
	}

	info, err := retry.Retry(ctx, c.retryConfig, func(ctx context.Context) (minio.UploadInfo, error) {
		info, err := c.client.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{
			ContentType:  opt.ContentType,
			UserMetadata: opt.Metadata,
		})
		if err != nil {
			return minio.UploadInfo{}, fmt.Errorf("s3: put object bucket=%s key=%s: %w", bucket, key, err)
		}
		return info, nil
	})
	if err != nil {
		return "", err
	}

	return info.ETag, nil
}

type getObjectResult struct {
	obj  io.ReadCloser
	info minio.ObjectInfo
}

func (c *S3Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, minio.ObjectInfo, error) {
	if bucket == "" {
		return nil, minio.ObjectInfo{}, fmt.Errorf("s3: bucket is required")
	}
	if key == "" {
		return nil, minio.ObjectInfo{}, fmt.Errorf("s3: key is required")
	}

	result, err := retry.Retry(ctx, c.retryConfig, func(ctx context.Context) (getObjectResult, error) {
		obj, err := c.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
		if err != nil {
			return getObjectResult{}, fmt.Errorf("s3: get object bucket=%s key=%s: %w", bucket, key, err)
		}

		// Important: GetObject returns a handle; errors can appear on Stat/Read.
		info, err := obj.Stat()
		if err != nil {
			_ = obj.Close()
			return getObjectResult{}, fmt.Errorf("s3: stat object bucket=%s key=%s: %w", bucket, key, err)
		}

		return getObjectResult{obj: obj, info: info}, nil
	})
	if err != nil {
		return nil, minio.ObjectInfo{}, inerr.NewErrNotFound(fmt.Sprintf("file with key %s in bucket %s", key, bucket))
	}

	return result.obj, result.info, nil
}

func (c *S3Client) StatObject(ctx context.Context, bucket, key string) (minio.ObjectInfo, error) {
	if bucket == "" {
		return minio.ObjectInfo{}, fmt.Errorf("s3: bucket is required")
	}
	if key == "" {
		return minio.ObjectInfo{}, fmt.Errorf("s3: key is required")
	}

	info, err := retry.Retry(ctx, c.retryConfig, func(ctx context.Context) (minio.ObjectInfo, error) {
		info, err := c.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
		if err != nil {
			return minio.ObjectInfo{}, fmt.Errorf("s3: stat object bucket=%s key=%s: %w", bucket, key, err)
		}
		return info, nil
	})
	if err != nil {
		return minio.ObjectInfo{}, inerr.NewErrNotFound(fmt.Sprintf("file with key %s in bucket %s", key, bucket))
	}
	return info, nil
}

func (c *S3Client) RemoveObject(ctx context.Context, bucket, key string) error {
	if bucket == "" {
		return fmt.Errorf("s3: bucket is required")
	}
	if key == "" {
		return fmt.Errorf("s3: key is required")
	}

	// S3 delete is idempotent: deleting missing key is typically "success".
	// minio RemoveObject returns error on request failure, not missing key.
	_, err := retry.Retry(ctx, c.retryConfig, func(ctx context.Context) (struct{}, error) {
		if err := c.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
			return struct{}{}, fmt.Errorf("s3: remove bucket=%s key=%s: %w", bucket, key, err)
		}
		return struct{}{}, nil
	})
	return err
}

func (c *S3Client) PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (*url.URL, error) {
	if bucket == "" {
		return nil, fmt.Errorf("s3: bucket is required")
	}
	if key == "" {
		return nil, fmt.Errorf("s3: key is required")
	}
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	u, err := retry.Retry(ctx, c.retryConfig, func(ctx context.Context) (*url.URL, error) {
		u, err := c.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
		if err != nil {
			return nil, fmt.Errorf("s3: presign get bucket=%s key=%s: %w", bucket, key, err)
		}
		return u, nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}
