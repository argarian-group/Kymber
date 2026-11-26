package backupstore

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"kymber/backend/internal/secrets"
)

type Config struct {
	Endpoint  string `json:"endpoint"`
	Region    string `json:"region"`
	Bucket    string `json:"bucket"`
	Prefix    string `json:"prefix"`
	AccessKey string `json:"access_key"`
}

type Store struct {
	client *minio.Client
	bucket string
	prefix string
}

func ReadConfig(ctx context.Context, db *sql.DB) (Config, []byte, error) {
	var config Config
	var encrypted []byte
	err := db.QueryRowContext(ctx, "SELECT s3_endpoint, s3_region, s3_bucket, s3_prefix, s3_access_key, encrypted_s3_secret FROM platform_settings WHERE id = 1").Scan(&config.Endpoint, &config.Region, &config.Bucket, &config.Prefix, &config.AccessKey, &encrypted)
	return config, encrypted, err
}

// Load returns nil when no destination is configured.
func Load(ctx context.Context, db *sql.DB) (*Store, error) {
	config, encrypted, err := ReadConfig(ctx, db)
	if err != nil || config.Bucket == "" {
		return nil, err
	}
	secret, err := secrets.Decrypt(encrypted)
	if err != nil {
		return nil, err
	}
	return New(config, string(secret))
}

func New(config Config, secret string) (*Store, error) {
	endpoint := config.Endpoint
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("endpoint must be a host such as s3.amazonaws.com or an http(s) URL without a path")
	}
	client, err := minio.New(parsed.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, secret, ""),
		Secure: parsed.Scheme == "https",
		Region: config.Region,
	})
	if err != nil {
		return nil, err
	}
	return &Store{client: client, bucket: config.Bucket, prefix: strings.Trim(config.Prefix, "/")}, nil
}

func (s *Store) Key(parts ...string) string {
	return path.Join(append([]string{s.prefix}, parts...)...)
}

func (s *Store) Put(ctx context.Context, key string, body io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: "application/gzip"})
	return err
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	if _, err := object.Stat(); err != nil {
		object.Close()
		return nil, err
	}
	return object, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// Check writes, reads back and removes a small object, which needs every permission backups use.
func (s *Store) Check(ctx context.Context) error {
	key := s.Key(".kymber-check")
	content := []byte("kymber")
	if err := s.Put(ctx, key, bytes.NewReader(content), int64(len(content))); err != nil {
		return fmt.Errorf("could not write to the bucket: %w", err)
	}
	object, err := s.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("could not read from the bucket: %w", err)
	}
	object.Close()
	if err := s.Delete(ctx, key); err != nil {
		return fmt.Errorf("could not delete from the bucket: %w", err)
	}
	return nil
}
