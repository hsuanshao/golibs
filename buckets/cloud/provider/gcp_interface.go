package provider

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

// GCSAPI defines the interface for interacting with Google Cloud Storage.
// This abstraction allows for mocking in unit tests.
type GCSAPI interface {
	// NewReader creates a new reader for the specified object.
	NewReader(ctx context.Context, bucket, object string) (io.ReadCloser, error)
	// NewWriter creates a new writer for the specified object.
	NewWriter(ctx context.Context, bucket, object string) (io.WriteCloser, error)
	// WriteObject writes data from reader to the specified object with content type and metadata.
	// This method is preferred over NewWriter when ContentType or Metadata must be set.
	WriteObject(ctx context.Context, bucket, object string, reader io.Reader, contentType string, metadata map[string]string) error
	// ObjectAttrs gets attributes of the specified object.
	ObjectAttrs(ctx context.Context, bucket, object string) (*storage.ObjectAttrs, error)
	// DeleteObject deletes the specified object.
	DeleteObject(ctx context.Context, bucket, object string) error
	// Objects lists objects in the bucket matching the query.
	Objects(ctx context.Context, bucket string, query *storage.Query) *storage.ObjectIterator
	// Close closes the client.
	Close() error
}

// gcsNativeClient implements GCSAPI using the official storage.Client.
type gcsNativeClient struct {
	client *storage.Client
}

func (c *gcsNativeClient) NewReader(ctx context.Context, bucket, object string) (io.ReadCloser, error) {
	return c.client.Bucket(bucket).Object(object).NewReader(ctx)
}

func (c *gcsNativeClient) NewWriter(ctx context.Context, bucket, object string) (io.WriteCloser, error) {
	return c.client.Bucket(bucket).Object(object).NewWriter(ctx), nil
}

func (c *gcsNativeClient) WriteObject(ctx context.Context, bucket, object string, reader io.Reader, contentType string, metadata map[string]string) error {
	wc := c.client.Bucket(bucket).Object(object).NewWriter(ctx)
	wc.ContentType = contentType
	wc.Metadata = metadata
	if _, err := io.Copy(wc, reader); err != nil {
		wc.Close()
		return err
	}
	return wc.Close()
}

func (c *gcsNativeClient) ObjectAttrs(ctx context.Context, bucket, object string) (*storage.ObjectAttrs, error) {
	return c.client.Bucket(bucket).Object(object).Attrs(ctx)
}

func (c *gcsNativeClient) DeleteObject(ctx context.Context, bucket, object string) error {
	return c.client.Bucket(bucket).Object(object).Delete(ctx)
}

func (c *gcsNativeClient) Objects(ctx context.Context, bucket string, query *storage.Query) *storage.ObjectIterator {
	return c.client.Bucket(bucket).Objects(ctx, query)
}

// Ensure interface compliance
func (c *gcsNativeClient) Close() error {
	return c.client.Close()
}
