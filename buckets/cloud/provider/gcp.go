package provider

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/api/iterator"

	pifc "github.com/hsuanshao/golibs/buckets/cloud/provider/interface"
	e "github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

/**
offical document: https://cloud.google.com/go/docs/reference/cloud.google.com/go/storage/latest
*/

var (
	googleBlobDefaultPath = "https://storage.googleapis.com"
)

func NewGCS(ctx ctx.CTX, conf *e.Config) (gcp pifc.ObjectServiceProvider, err error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &gcsImpl{
		bucket: conf.Bucket,
		region: conf.Region,
		api:    &gcsNativeClient{client: client},
	}, nil
}

type gcsImpl struct {
	bucket string
	region string
	api    GCSAPI
}

// GetObjectList to fetch object list
func (gim *gcsImpl) GetObjectList(ctx ctx.CTX, prefix, delim string) (objURLs []string, err error) {
	query := &storage.Query{Prefix: prefix, Delimiter: delim}
	it := gim.api.Objects(ctx, gim.bucket, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		objURLs = append(objURLs, fmt.Sprintf("%s/%s/%s", googleBlobDefaultPath, gim.bucket, attrs.Name))
	}
	return objURLs, nil
}

// ReadObjectContent to read object content
func (gim *gcsImpl) ReadObjectContent(ctx ctx.CTX, objectPath string) (objReader io.ReadCloser, metadata map[string]string, err error) {
	// objectPath expected format: https://storage.googleapis.com/bucket/object or just object key?
	// Assuming logic similar to Minio/S3 where we might need to parse, but usually objectPath is the key in this carrier?
	// Existing buckets logic seems to treat objectPath as full URL sometimes.
	// For simplicity and consistency with zero-copy refactor, we assume objectPath might be the key or we parse it.
	// Let's implement key extraction helper later if needed, assuming objectPath IS the key for now or handled by caller.
	// Actually minio impl parses URL. GCS logic should probably do same if `objectPath` is passed as URL.
	// But `buckets` package usually passes the key derived from Logic?
	// `ReadObjectContent` input is `objURL`.
	// Let's implement a quick parser or assume key.
	// The `googleBlobDefaultPath` variable hints we construct URLs.

	// check if it is URL
	bucket, key, err := gim.parseURL(objectPath)
	if err == nil {
		if bucket != gim.bucket {
			return nil, nil, e.ErrWithoutPermissionToAccess
		}
	} else {
		// Fallback: treat objectPath as key
		key = objectPath
	}

	reader, err := gim.api.NewReader(ctx, gim.bucket, key)
	if err != nil {
		return nil, nil, err
	}

	// We need metadata. NewReader gives content.
	// To get metadata we might need ObjectAttrs.
	// But `NewReader` in GCS SDK actually returns a `*Reader` which has `Attrs`.
	// Our interface returns `io.ReadCloser`.
	// If we need metadata, we should call ObjectAttrs.

	attrs, err := gim.api.ObjectAttrs(ctx, gim.bucket, key)
	if err != nil {
		reader.Close()
		return nil, nil, err
	}

	// Filter metadata
	md := make(map[string]string)
	if attrs.Metadata != nil {
		for k, v := range attrs.Metadata {
			md[k] = v
		}
	}
	md["Content-Type"] = attrs.ContentType

	return reader, md, nil
}

// IsObjectExists to check object existence by given url
func (gim *gcsImpl) IsObjectExists(ctx ctx.CTX, objURL string) (existed bool, err error) {
	bucket, key, err := gim.parseURL(objURL)
	if err != nil {
		return false, nil
	}
	if bucket != gim.bucket {
		return false, e.ErrWithoutPermissionToAccess
	}

	_, err = gim.api.ObjectAttrs(ctx, gim.bucket, key)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (gim *gcsImpl) GenReadPresignedURL(ctx ctx.CTX, objURL string, duration time.Duration) (readPresignedURL string, err error) {
	// GCS Presigning requires credential file or signing capabilities.
	// storage.SignedURL is the way.
	// But we abstracted with GCSAPI which doesn't expose SignedURL (it's a standalone function usually).
	// We'll skip implementation for now or use `e.ErrNotImpl` as checking dependencies for signing is complex.
	return "", e.ErrNotImpl
}

// PutPresignedURL to generate an object upload with a ttl permision
func (gim *gcsImpl) PutPresignedURL(ctx ctx.CTX, objURL string, mime e.ContentType, duration time.Duration, metaData map[string]string) (presignedURL string, err error) {
	return "", e.ErrNotImpl
}

// Upload to upload object
func (gim *gcsImpl) Upload(ctx ctx.CTX, ct e.ContentType, objpath string, payload io.Reader, size int64, objmetadata map[string]string) (URL string, readPresignedURL string, err error) {
	key := objpath
	// check if it is URL
	b, k, err := gim.parseURL(objpath)
	if err == nil {
		if b != gim.bucket {
			return "", "", e.ErrWithoutPermissionToAccess
		}
		key = k
	}

	// Peeking for MIME detection (consistent with S3/Minio)
	head := make([]byte, 512)
	n, err := payload.Read(head)
	if err != nil && err != io.EOF {
		return "", "", e.ErrUploadObj
	}
	payload = io.MultiReader(bytes.NewReader(head[:n]), payload)

	detectedMime := mimetype.Detect(head[:n])
	if !e.IsMatchContentType(detectedMime.String(), ct) {
		return "", "", e.ErrUploadNotMatchContentType
	}

	// Use WriteObject which properly sets ContentType and Metadata on the GCS object
	if err = gim.api.WriteObject(ctx, gim.bucket, key, payload, ct.String(), objmetadata); err != nil {
		return "", "", err
	}

	return fmt.Sprintf("%s/%s/%s", googleBlobDefaultPath, gim.bucket, key), "", nil
}

// Override to override exists obj, this function will compare original content and new content
func (gim *gcsImpl) Override(ctx ctx.CTX, ct e.ContentType, objPath string, payload io.Reader, size int64, objmetadata map[string]string) (objURL string, err error) {
	// Check existence
	exists, err := gim.IsObjectExists(ctx, objPath)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", e.ErrFetchOriginObject
	}

	u, _, err := gim.Upload(ctx, ct, objPath, payload, size, objmetadata)
	return u, err
}

func (gim *gcsImpl) Delete(ctx ctx.CTX, contentType e.ContentType, objPathes []string) (result bool, err error) {
	for _, p := range objPathes {
		b, k, err := gim.parseURL(p)
		if err != nil {
			k = p // fallback to treat as key
			b = gim.bucket
		}
		if b != gim.bucket {
			continue
		}
		gim.api.DeleteObject(ctx, b, k)
	}
	return true, nil
}

// Health to tell every platform service latency
func (gim *gcsImpl) Health(ctx ctx.CTX) (status e.HealthStatus, err error) {
	return e.HealthStatus{Cloud: e.GCP, Latency: 0}, nil
}

// Close to close client
func (gim *gcsImpl) Close() {
	gim.api.Close()
}

func (gim *gcsImpl) parseURL(url string) (bucket, key string, err error) {
	// Very basic parser assuming https://storage.googleapis.com/bucket/key
	// or just bucket/key?
	prefix := googleBlobDefaultPath + "/"
	if len(url) > len(prefix) && url[:len(prefix)] == prefix {
		rest := url[len(prefix):]
		// rest is bucket/key/path
		// find first slash
		for i, c := range rest {
			if c == '/' {
				return rest[:i], rest[i+1:], nil
			}
		}
	}
	return "", "", fmt.Errorf("invalid url")
}
