package s3store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

func TestConfigValidationAndPrefixNormalization(t *testing.T) {
	for _, test := range []struct {
		config Config
		want   string
	}{
		{Config{Region: "us-east-1"}, "bucket"},
		{Config{Bucket: "bucket"}, "region"},
		{Config{Bucket: "bucket", Region: "us-east-1", Prefix: "/bad"}, "prefix"},
		{Config{Bucket: "bucket", Region: "us-east-1", Prefix: "bad//key"}, "prefix"},
		{Config{Bucket: "bucket", Region: "us-east-1", Prefix: "bad/../key"}, "prefix"},
	} {
		if _, err := test.config.Validate(); err == nil || !strings.Contains(strings.ToLower(err.Error()), test.want) {
			t.Fatalf("Validate(%+v) = %v, want %q", test.config, err, test.want)
		}
	}
	config, err := (Config{Bucket: " bucket ", Region: " us-east-1 ", Prefix: "vaults/team"}).Validate()
	if err != nil || config.Bucket != "bucket" || config.Region != "us-east-1" || config.Prefix != "vaults/team" {
		t.Fatalf("validated config = %+v, %v", config, err)
	}
}

func TestReadListPaginationAndFailures(t *testing.T) {
	fake := newFakeClient()
	fake.objects["root/a"] = []byte("a")
	fake.objects["root/b"] = []byte("b")
	store, err := newWithClient(Config{Bucket: "bucket", Region: "region", Prefix: "root"}, fake)
	if err != nil {
		t.Fatal(err)
	}
	data, etag, err := store.Read(context.Background(), "a")
	if err != nil || string(data) != "a" || etag == "" {
		t.Fatalf("read = %q, %q, %v", data, etag, err)
	}
	if _, _, err := store.Read(context.Background(), "missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing error = %v", err)
	}
	objects, err := store.List(context.Background(), "", 2)
	if err != nil || len(objects) != 2 || objects[0].Key != "a" || objects[1].Key != "b" || fake.listCalls != 2 {
		t.Fatalf("list = %+v, calls %d, err %v", objects, fake.listCalls, err)
	}
	fake.listErr = apiError{code: "AccessDenied"}
	if _, err := store.List(context.Background(), "", 2); err == nil || !strings.Contains(err.Error(), "AccessDenied") {
		t.Fatalf("list failure = %v", err)
	}
}

func TestConditionalWrites(t *testing.T) {
	fake := newFakeClient()
	store, _ := newWithClient(Config{Bucket: "bucket", Region: "region"}, fake)
	created, etag, err := store.Create(context.Background(), "record", []byte("one"))
	if err != nil || !created || etag == "" {
		t.Fatalf("create = %t, %q, %v", created, etag, err)
	}
	created, repeatedETag, err := store.Create(context.Background(), "record", []byte("one"))
	if err != nil || created || repeatedETag != etag {
		t.Fatalf("retry = %t, %q, %v", created, repeatedETag, err)
	}
	if _, _, err := store.Create(context.Background(), "record", []byte("other")); !IsConflict(err) {
		t.Fatalf("collision = %v", err)
	}
	replacementETag, err := store.Replace(context.Background(), "record", []byte("two"), etag)
	if err != nil || replacementETag == etag || string(fake.objects["record"]) != "two" {
		t.Fatalf("replace = %q, %v", replacementETag, err)
	}
	if _, err := store.Replace(context.Background(), "record", []byte("three"), etag); !IsConflict(err) {
		t.Fatalf("concurrent replace = %v", err)
	}
}

type apiError struct{ code string }

func (e apiError) Error() string                 { return e.code }
func (e apiError) ErrorCode() string             { return e.code }
func (e apiError) ErrorMessage() string          { return e.code }
func (e apiError) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

type fakeClient struct {
	objects   map[string][]byte
	etags     map[string]string
	nextETag  int
	listCalls int
	listErr   error
}

func newFakeClient() *fakeClient {
	return &fakeClient{objects: map[string][]byte{}, etags: map[string]string{}, nextETag: 1}
}

func (f *fakeClient) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	data, ok := f.objects[*input.Key]
	if !ok {
		return nil, apiError{code: "NoSuchKey"}
	}
	etag := f.etags[*input.Key]
	if etag == "" {
		etag = f.newETag()
		f.etags[*input.Key] = etag
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(data)), ETag: &etag}, nil
}

func (f *fakeClient) ListObjectsV2(_ context.Context, input *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.listCalls++
	keys := []string{}
	for key := range f.objects {
		if strings.HasPrefix(key, *input.Prefix) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	start := 0
	if input.ContinuationToken != nil {
		start = 1
	}
	if start >= len(keys) {
		return &s3.ListObjectsV2Output{}, nil
	}
	key := keys[start]
	size := int64(len(f.objects[key]))
	etag := f.etags[key]
	truncated := start+1 < len(keys)
	result := &s3.ListObjectsV2Output{Contents: []types.Object{{Key: &key, ETag: &etag, Size: &size}}, IsTruncated: &truncated}
	if truncated {
		token := "next"
		result.NextContinuationToken = &token
	}
	return result, nil
}

func (f *fakeClient) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	key := *input.Key
	if input.IfNoneMatch != nil {
		if _, exists := f.objects[key]; exists {
			return nil, apiError{code: "PreconditionFailed"}
		}
	}
	if input.IfMatch != nil && f.etags[key] != *input.IfMatch {
		return nil, apiError{code: "PreconditionFailed"}
	}
	data, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	etag := f.newETag()
	f.objects[key], f.etags[key] = data, etag
	return &s3.PutObjectOutput{ETag: &etag}, nil
}

func (f *fakeClient) newETag() string {
	f.nextETag++
	return fmt.Sprintf("etag-%d", f.nextETag)
}
