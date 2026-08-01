package evidence

import (
	"context"
	"time"

	"gnosis/internal/s3store"
)

type objectStore interface {
	Read(context.Context, string) ([]byte, string, error)
	List(context.Context, string, int) ([]s3store.Object, error)
	Create(context.Context, string, []byte) (bool, string, error)
	Replace(context.Context, string, []byte, string) (string, error)
	Location() string
}

type S3Store = Store

func NewS3(ctx context.Context, config s3store.Config) (*S3Store, error) {
	objects, err := s3store.New(ctx, config)
	if err != nil {
		return nil, err
	}
	return newS3(objects), nil
}

func newS3(objects objectStore) *S3Store {
	return &Store{objects: objects, now: time.Now}
}
