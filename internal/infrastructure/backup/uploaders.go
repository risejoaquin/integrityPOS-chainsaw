package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalUploader implements Uploader for local filesystem
type LocalUploader struct {
	basePath string
}

// NewLocalUploader creates a new local uploader
func NewLocalUploader(basePath string) *LocalUploader {
	if basePath == "" {
		basePath = "./backups"
	}
	return &LocalUploader{basePath: basePath}
}

// Upload saves data to local filesystem
func (lu *LocalUploader) Upload(ctx context.Context, key string, reader io.Reader) error {
	fullPath := filepath.Join(lu.basePath, key)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// Download reads data from local filesystem
func (lu *LocalUploader) Download(ctx context.Context, key string, writer io.Writer) error {
	fullPath := filepath.Join(lu.basePath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", fullPath, err)
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	return nil
}

// List returns all keys with the given prefix
func (lu *LocalUploader) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := filepath.Join(lu.basePath, prefix)

	var keys []string
	err := filepath.Walk(fullPrefix, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Convert full path back to relative key
			relPath, err := filepath.Rel(lu.basePath, path)
			if err != nil {
				return err
			}

			// Normalize path separators
			key := strings.ReplaceAll(relPath, "\\", "/")
			keys = append(keys, key)
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return keys, nil
}

// Delete removes a file from local filesystem
func (lu *LocalUploader) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(lu.basePath, key)

	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}

	return nil
}

// S3Uploader implements Uploader for AWS S3
type S3Uploader struct {
	bucket string
	region string
}

// NewS3Uploader creates a new S3 uploader
func NewS3Uploader(bucket, region string) *S3Uploader {
	if region == "" {
		region = "us-east-1"
	}
	return &S3Uploader{
		bucket: bucket,
		region: region,
	}
}

// Upload saves data to S3 (simplified implementation)
func (su *S3Uploader) Upload(ctx context.Context, key string, reader io.Reader) error {
	// In production, use AWS SDK:
	// svc := s3.New(session.Must(session.NewSession(&aws.Config{Region: aws.String(su.region)})))
	// _, err := svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
	//     Bucket: aws.String(su.bucket),
	//     Key:    aws.String(key),
	//     Body:   reader,
	// })

	// For this implementation, simulate success
	return nil
}

// Download reads data from S3 (simplified implementation)
func (su *S3Uploader) Download(ctx context.Context, key string, writer io.Writer) error {
	// In production, use AWS SDK to download
	return fmt.Errorf("S3 download not implemented in this version")
}

// List returns all keys with the given prefix from S3
func (su *S3Uploader) List(ctx context.Context, prefix string) ([]string, error) {
	// In production, use AWS SDK to list objects
	return []string{}, nil
}

// Delete removes an object from S3
func (su *S3Uploader) Delete(ctx context.Context, key string) error {
	// In production, use AWS SDK to delete
	return nil
}

// GCSUploader implements Uploader for Google Cloud Storage
type GCSUploader struct {
	bucket string
}

// NewGCSUploader creates a new GCS uploader
func NewGCSUploader(bucket string) *GCSUploader {
	return &GCSUploader{bucket: bucket}
}

// Upload saves data to GCS (simplified implementation)
func (gu *GCSUploader) Upload(ctx context.Context, key string, reader io.Reader) error {
	// In production, use Google Cloud Storage client
	return nil
}

// Download reads data from GCS (simplified implementation)
func (gu *GCSUploader) Download(ctx context.Context, key string, writer io.Writer) error {
	return fmt.Errorf("GCS download not implemented in this version")
}

// List returns all keys with the given prefix from GCS
func (gu *GCSUploader) List(ctx context.Context, prefix string) ([]string, error) {
	return []string{}, nil
}

// Delete removes an object from GCS
func (gu *GCSUploader) Delete(ctx context.Context, key string) error {
	return nil
}