// Package s3store provides the bounded S3 object operations used by gnosis.
package s3store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const MaxListObjects = 10_000

// Config identifies one S3 namespace. Credentials come from the AWS SDK chain.
type Config struct {
	Bucket string
	Region string
	Prefix string
}

func (c Config) Validate() (Config, error) {
	c.Bucket = strings.TrimSpace(c.Bucket)
	c.Region = strings.TrimSpace(c.Region)
	if c.Bucket == "" {
		return Config{}, fmt.Errorf("S3 bucket must not be empty")
	}
	if c.Region == "" {
		return Config{}, fmt.Errorf("S3 region must not be empty")
	}
	prefix, err := NormalizePrefix(c.Prefix)
	if err != nil {
		return Config{}, err
	}
	c.Prefix = prefix
	return c, nil
}

func NormalizePrefix(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") {
		return "", fmt.Errorf("S3 prefix must not start or end with /")
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("S3 prefix contains an unsafe path segment")
		}
	}
	return value, nil
}

type client interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type Store struct {
	config Config
	client client
}

func New(ctx context.Context, config Config) (*Store, error) {
	config, err := config.Validate()
	if err != nil {
		return nil, err
	}
	loaded, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	return newWithClient(config, s3.NewFromConfig(loaded))
}

func newWithClient(config Config, client client) (*Store, error) {
	config, err := config.Validate()
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("S3 client must not be nil")
	}
	return &Store{config: config, client: client}, nil
}

func (s *Store) Location() string {
	if s.config.Prefix == "" {
		return "s3://" + s.config.Bucket
	}
	return "s3://" + s.config.Bucket + "/" + s.config.Prefix
}

type Object struct {
	Key  string
	ETag string
	Size int64
}

func (s *Store) Read(ctx context.Context, key string) ([]byte, string, error) {
	full, err := s.key(key)
	if err != nil {
		return nil, "", err
	}
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.config.Bucket, Key: &full})
	if isCode(err, "NoSuchKey", "NotFound") {
		return nil, "", fmt.Errorf("read S3 object %q: %w", key, fs.ErrNotExist)
	}
	if err != nil {
		return nil, "", fmt.Errorf("read S3 object %q: %w", key, err)
	}
	defer result.Body.Close()
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read S3 object %q body: %w", key, err)
	}
	return data, strings.TrimSpace(value(result.ETag)), nil
}

func (s *Store) List(ctx context.Context, prefix string, limit int) ([]Object, error) {
	if limit < 1 || limit > MaxListObjects {
		return nil, fmt.Errorf("S3 list limit must be between 1 and %d", MaxListObjects)
	}
	full, err := s.prefix(prefix)
	if err != nil {
		return nil, err
	}
	objects := make([]Object, 0, limit)
	var token *string
	for len(objects) < limit {
		remaining := int32(limit - len(objects))
		result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: &s.config.Bucket, Prefix: &full, ContinuationToken: token, MaxKeys: &remaining,
		})
		if err != nil {
			return nil, fmt.Errorf("list S3 prefix %q: %w", prefix, err)
		}
		for _, item := range result.Contents {
			key := strings.TrimPrefix(value(item.Key), s.basePrefix())
			objects = append(objects, Object{Key: key, ETag: value(item.ETag), Size: valueInt64(item.Size)})
			if len(objects) == limit {
				break
			}
		}
		if !valueBool(result.IsTruncated) || result.NextContinuationToken == nil {
			break
		}
		token = result.NextContinuationToken
	}
	sort.Slice(objects, func(i, j int) bool { return objects[i].Key < objects[j].Key })
	return objects, nil
}

// Create writes only an absent key. An identical retry is unchanged.
func (s *Store) Create(ctx context.Context, key string, data []byte) (bool, string, error) {
	full, err := s.key(key)
	if err != nil {
		return false, "", err
	}
	star := "*"
	result, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.config.Bucket, Key: &full, Body: bytes.NewReader(data), IfNoneMatch: &star,
	})
	if err == nil {
		return true, value(result.ETag), nil
	}
	if !isConflict(err) {
		return false, "", fmt.Errorf("create S3 object %q: %w", key, err)
	}
	existing, etag, readErr := s.Read(ctx, key)
	if readErr == nil && bytes.Equal(existing, data) {
		return false, etag, nil
	}
	return false, "", Conflict{Key: key}
}

func (s *Store) Replace(ctx context.Context, key string, data []byte, etag string) (string, error) {
	full, err := s.key(key)
	if err != nil {
		return "", err
	}
	etag = strings.TrimSpace(etag)
	if etag == "" {
		return "", fmt.Errorf("replace S3 object %q: ETag must not be empty", key)
	}
	result, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.config.Bucket, Key: &full, Body: bytes.NewReader(data), IfMatch: &etag,
	})
	if isConflict(err) {
		return "", Conflict{Key: key}
	}
	if err != nil {
		return "", fmt.Errorf("replace S3 object %q: %w", key, err)
	}
	return value(result.ETag), nil
}

type Conflict struct{ Key string }

func (e Conflict) Error() string { return fmt.Sprintf("S3 object %q changed concurrently", e.Key) }

func IsConflict(err error) bool {
	var conflict Conflict
	return errors.As(err, &conflict)
}

func (s *Store) key(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" || strings.HasPrefix(key, "/") || path.Clean(key) != key || strings.Contains(key, "//") {
		return "", fmt.Errorf("unsafe S3 object key %q", key)
	}
	return s.basePrefix() + key, nil
}

func (s *Store) prefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return s.basePrefix(), nil
	}
	if strings.HasPrefix(prefix, "/") || path.Clean(prefix) != strings.TrimSuffix(prefix, "/") || strings.Contains(prefix, "//") {
		return "", fmt.Errorf("unsafe S3 object prefix %q", prefix)
	}
	return s.basePrefix() + prefix, nil
}

func (s *Store) basePrefix() string {
	if s.config.Prefix == "" {
		return ""
	}
	return s.config.Prefix + "/"
}

func isConflict(err error) bool {
	return isCode(err, "PreconditionFailed", "ConditionalRequestConflict")
}

func isCode(err error, codes ...string) bool {
	var api smithy.APIError
	if !errors.As(err, &api) {
		return false
	}
	for _, code := range codes {
		if api.ErrorCode() == code {
			return true
		}
	}
	return false
}

func value(pointer *string) string {
	if pointer == nil {
		return ""
	}
	return *pointer
}

func valueBool(pointer *bool) bool {
	return pointer != nil && *pointer
}

func valueInt64(pointer *int64) int64 {
	if pointer == nil {
		return 0
	}
	return *pointer
}
