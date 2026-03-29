package ifc

import (
	"io"
	"time"

	"github.com/hsuanshao/golibs/ctx"
)

// BucketReader provide methods to access objects from clouds
type BucketReader interface {
	// ListObjURLs to list out objects storage URLs by given prefix and delim
	ListObjURLs(ctx ctx.CTX, prefix, delim string) (objURLs []string, err error)
	// ReadObjectContent to get object content by given object url
	// It returns an io.ReadCloser which the caller MUST close.
	ReadObjectContent(ctx ctx.CTX, objURL string) (objReader io.ReadCloser, metadata map[string]string, err error)
	// CheckObjExists to check if the given blob path exists
	CheckObjExists(ctx ctx.CTX, objURL string) (exists bool, err error)
	// GenerateReadPresignedURL to generate a "read" permision within a "period" (by given duration),
	// with a specific object storage path, if given object path/URL not fould it will return an error
	GenerateReadPresignedURL(ctx ctx.CTX, duration time.Duration, objURL string) (presignedURL string, err error)
	// Close to Close BucketReader agent
	Close()
}
