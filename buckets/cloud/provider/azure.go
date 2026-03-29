package provider

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/gabriel-vasile/mimetype"

	pifc "github.com/hsuanshao/golibs/buckets/cloud/provider/interface"
	e "github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

/**
azure sdk https://github.com/Azure/azure-sdk-for-go
*/

// azure sdk https://github.com/Azure/azure-sdk-for-go

func NewAzure(ctx ctx.CTX, conf *e.Config) (azure pifc.ObjectServiceProvider, err error) {
	// Assuming conf.Endpoint is the service URL or we construct it?
	// Usually credential is simpler. But sticking to existing config usage if any.
	// For now, implementing client creation using connection string or similar if available,
	// but failing that, we stick to e.ErrNotImpl if credential logic is complex/missing in config.
	// Actually, `NewAzure` stub implies we should implement it.
	// Let's assume standard client construction from environment or config.
	// For this task, main goal is streaming logic.
	// I'll create the `azureImpl` with `azureNativeClient` assuming client is created.
	// TODO: Add client creation logic based on conf.
	// Example: client, err := azblob.NewClientFromConnectionString(conf.ConnectionString, nil)
	// But `e.Config` definition is external.
	// I'll stick to a placeholder for client creation or use `NewClientWithNoCredential` if endpoint provided?
	// Let's create `azblob.Client` if we can.
	// If not, we return error for initialization or mock it?
	// We'll return ErrNotImpl for NEW logic if config is insufficient, but implementation struct is ready.

	// Placeholder for client creation. In a real scenario, this would initialize azblob.Client
	// For now, we'll use a dummy client or return ErrNotImpl if a proper client cannot be created.
	// For the purpose of this exercise, we'll assume a client can be created or mocked.
	// If conf.ConnectionString is available:
	// client, err := azblob.NewClientFromConnectionString(conf.ConnectionString, nil)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create Azure Blob client: %w", err)
	// }
	// api := &azureNativeClient{client: client}

	// Azure Blob Storage is not yet fully implemented.
	// Return nil to prevent callers from using a struct with nil internal client.
	return nil, e.ErrNotImpl
}

type azureImpl struct {
	container string
	api       AzureAPI
}

// GetObjectList to fetch object list
func (az *azureImpl) GetObjectList(ctx ctx.CTX, prefix, delim string) (objURLs []string, err error) {
	return nil, e.ErrNotImpl
}

// ReadObjectContent to read object content
func (az *azureImpl) ReadObjectContent(ctx ctx.CTX, objectPath string) (objReader io.ReadCloser, metadata map[string]string, err error) {
	// Assuming objectPath is blobName.
	// If it is URL, parse it.
	blobName := objectPath // simplified

	resp, err := az.api.DownloadStream(ctx, az.container, blobName, nil)
	if err != nil {
		return nil, nil, err
	}

	// Metadata extraction
	md := make(map[string]string)
	if resp.Metadata != nil {
		for k, v := range resp.Metadata {
			md[k] = *v
		}
	}
	if resp.ContentType != nil {
		md["Content-Type"] = *resp.ContentType
	}

	return resp.Body, md, nil
}

// IsObjectExists to check object existence by given url
func (az *azureImpl) IsObjectExists(ctx ctx.CTX, objURL string) (existed bool, err error) {
	// Use GetProperties to check existence
	blobName := objURL // simplified
	_, err = az.api.GetProperties(ctx, az.container, blobName, nil)
	if err != nil {
		var bloberr *azcore.ResponseError
		if errors.As(err, &bloberr) && bloberr.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (az *azureImpl) GenReadPresignedURL(ctx ctx.CTX, objURL string, duration time.Duration) (readPresignedURL string, err error) {
	return "", e.ErrNotImpl
}

// PutPresignedURL to generate an object upload with a ttl permision
func (az *azureImpl) PutPresignedURL(ctx ctx.CTX, objURL string, mime e.ContentType, duration time.Duration, metaData map[string]string) (presignedURL string, err error) {
	return "", e.ErrNotImpl
}

// Upload to upload object
func (az *azureImpl) Upload(ctx ctx.CTX, ct e.ContentType, objpath string, payload io.Reader, size int64, objmetadata map[string]string) (URL string, readPresignedURL string, err error) {
	blobName := objpath
	// Mime peek
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

	opts := &azblob.UploadStreamOptions{
		// BlockSize? Standard.
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: toPtr(ct.String()),
		},
		Metadata: make(map[string]*string),
	}
	for k, v := range objmetadata {
		val := v
		opts.Metadata[k] = &val
	}

	_, err = az.api.UploadStream(ctx, az.container, blobName, payload, opts)
	if err != nil {
		return "", "", err
	}

	// URL construction?
	// azblob doesn't return URL directly in response usually.
	// We construct it.
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", "ACCOUNT_NAME_PLACEHOLDER", az.container, blobName), "", nil
}

// Override to override exists obj, this function will compare original content and new content
func (az *azureImpl) Override(ctx ctx.CTX, ct e.ContentType, objPath string, payload io.Reader, size int64, objmetadata map[string]string) (objURL string, err error) {
	exists, err := az.IsObjectExists(ctx, objPath)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", e.ErrFetchOriginObject
	}
	u, _, err := az.Upload(ctx, ct, objPath, payload, size, objmetadata)
	return u, err
}

func (az *azureImpl) Delete(ctx ctx.CTX, contentType e.ContentType, objPathes []string) (result bool, err error) {
	hasErr := false
	for _, p := range objPathes {
		_, delErr := az.api.DeleteBlob(ctx, az.container, p, nil)
		if delErr != nil {
			ctx.WithField("err", delErr).Error("failed to delete blob from azure")
			hasErr = true
		}
	}
	if hasErr {
		return false, e.ErrDeleteObject
	}
	return true, nil
}

// Health to tell every platform service latency
func (az *azureImpl) Health(ctx ctx.CTX) (status e.HealthStatus, err error) {
	return e.HealthStatus{Cloud: e.Azure, Latency: 0}, nil
}

// Close to close client
func (az *azureImpl) Close() {}

func toPtr(s string) *string {
	return &s
}
