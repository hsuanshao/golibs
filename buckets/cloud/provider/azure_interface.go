package provider

import (
	"context"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
)

// AzureAPI defines the interface for Azure Cloud Storage interaction.
type AzureAPI interface {
	UploadStream(ctx context.Context, containerName, blobName string, body io.Reader, options *azblob.UploadStreamOptions) (azblob.UploadStreamResponse, error)
	DownloadStream(ctx context.Context, containerName, blobName string, options *azblob.DownloadStreamOptions) (azblob.DownloadStreamResponse, error)
	DeleteBlob(ctx context.Context, containerName, blobName string, options *azblob.DeleteBlobOptions) (azblob.DeleteBlobResponse, error)
	GetProperties(ctx context.Context, containerName, blobName string, options *blob.GetPropertiesOptions) (blob.GetPropertiesResponse, error)
}

// azureNativeClient implements AzureAPI using azblob.Client
type azureNativeClient struct {
	client *azblob.Client
}

func (c *azureNativeClient) UploadStream(ctx context.Context, containerName, blobName string, body io.Reader, options *azblob.UploadStreamOptions) (azblob.UploadStreamResponse, error) {
	return c.client.UploadStream(ctx, containerName, blobName, body, options)
}

func (c *azureNativeClient) DownloadStream(ctx context.Context, containerName, blobName string, options *azblob.DownloadStreamOptions) (azblob.DownloadStreamResponse, error) {
	return c.client.DownloadStream(ctx, containerName, blobName, options)
}

func (c *azureNativeClient) DeleteBlob(ctx context.Context, containerName, blobName string, options *azblob.DeleteBlobOptions) (azblob.DeleteBlobResponse, error) {
	return c.client.DeleteBlob(ctx, containerName, blobName, options)
}

func (c *azureNativeClient) GetProperties(ctx context.Context, containerName, blobName string, options *blob.GetPropertiesOptions) (blob.GetPropertiesResponse, error) {
	// azblob.Client.ServiceClient() -> ServiceClient
	// ServiceClient.NewContainerClient -> ContainerClient
	// ContainerClient.NewBlobClient -> BlobClient
	// BlobClient.GetProperties -> (blob.GetPropertiesResponse, error)
	return c.client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName).GetProperties(ctx, options)
}
