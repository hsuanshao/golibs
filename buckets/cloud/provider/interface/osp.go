package ifc

import (
	"io"
	"time"

	e "github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

// ObjectServiceProvider describe methods that need to implment from native cloud SDK
type ObjectServiceProvider interface {
	// GetObjectList to fetch object list
	GetObjectList(ctx ctx.CTX, prefix, delim string) (objURLs []string, err error)
	// ReadObjectContent to read object content
	ReadObjectContent(ctx ctx.CTX, objectPath string) (objReader io.ReadCloser, metadata map[string]string, err error)
	// IsObjectExists to check object existence by given url
	IsObjectExists(ctx ctx.CTX, objURL string) (existed bool, err error)
	// GenReadPresginedURL to generate a private object with ttl permission
	GenReadPresignedURL(ctx ctx.CTX, objURL string, duration time.Duration) (readPresignedURL string, err error)
	// PutPresignedURL to generate an object upload with a ttl permision
	PutPresignedURL(ctx ctx.CTX, objURL string, mime e.ContentType, duration time.Duration, metaData map[string]string) (presignedURL string, err error)
	// Upload to upload object
	Upload(ctx ctx.CTX, ct e.ContentType, objpath string, payload io.Reader, size int64, objmetadata map[string]string) (URL string, readPresignedURL string, err error)
	// Override to override exists obj, this function will compare original content and new content
	Override(ctx ctx.CTX, ct e.ContentType, objPath string, payload io.Reader, size int64, objmetadata map[string]string) (objURL string, err error)
	// Delete for delete object item from blob object storage
	Delete(ctx ctx.CTX, contentType e.ContentType, objPathes []string) (result bool, err error)
	// Health to tell every platform service latency
	Health(ctx ctx.CTX) (status e.HealthStatus, err error)
	// Close to close client
	Close()
}
