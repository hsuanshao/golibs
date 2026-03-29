package provider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/gabriel-vasile/mimetype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	e "github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

// MockAzureClient for testing
type MockAzureClient struct {
	mock.Mock
}

func (m *MockAzureClient) UploadStream(ctx context.Context, containerName, blobName string, body io.Reader, options *azblob.UploadStreamOptions) (azblob.UploadStreamResponse, error) {
	args := m.Called(ctx, containerName, blobName, body, options)

	// Read body to verify content?
	if body != nil {
		io.ReadAll(body)
	}

	return args.Get(0).(azblob.UploadStreamResponse), args.Error(1)
}

func (m *MockAzureClient) DownloadStream(ctx context.Context, containerName, blobName string, options *azblob.DownloadStreamOptions) (azblob.DownloadStreamResponse, error) {
	args := m.Called(ctx, containerName, blobName, options)
	return args.Get(0).(azblob.DownloadStreamResponse), args.Error(1)
}

func (m *MockAzureClient) DeleteBlob(ctx context.Context, containerName, blobName string, options *azblob.DeleteBlobOptions) (azblob.DeleteBlobResponse, error) {
	args := m.Called(ctx, containerName, blobName, options)
	return args.Get(0).(azblob.DeleteBlobResponse), args.Error(1)
}

func (m *MockAzureClient) GetProperties(ctx context.Context, containerName, blobName string, options *blob.GetPropertiesOptions) (blob.GetPropertiesResponse, error) {
	args := m.Called(ctx, containerName, blobName, options)
	return args.Get(0).(blob.GetPropertiesResponse), args.Error(1)
}

type AzureTestSuite struct {
	suite.Suite
	mockClient *MockAzureClient
	provider   *azureImpl
	ctx        ctx.CTX
}

func TestAzureTestSuite(t *testing.T) {
	suite.Run(t, new(AzureTestSuite))
}

func (s *AzureTestSuite) SetupTest() {
	s.mockClient = new(MockAzureClient)
	s.ctx = ctx.Background()
	s.provider = &azureImpl{
		container: "test-container",
		api:       s.mockClient,
	}
}

func (s *AzureTestSuite) TestUpload() {
	content := []byte("test content")
	reader := bytes.NewReader(content)

	s.mockClient.On("UploadStream", mock.Anything, "test-container", "test/file.txt", mock.Anything, mock.Anything).Return(azblob.UploadStreamResponse{}, nil)

	// Determine mime type dynamically
	detectedMime := mimetype.Detect(content).String()

	url, _, err := s.provider.Upload(s.ctx, e.ContentType(detectedMime), "test/file.txt", reader, int64(len(content)), nil)

	s.Require().NoError(err)
	s.Contains(url, "test-container/test/file.txt")
}

func (s *AzureTestSuite) TestReadObjectContent() {
	content := "file content"
	bodyRC := io.NopCloser(bytes.NewReader([]byte(content)))

	expectedType := "text/plain"
	resp := azblob.DownloadStreamResponse{
		DownloadResponse: blob.DownloadResponse{
			Body:        bodyRC,
			ContentType: &expectedType,
		},
	}

	s.mockClient.On("DownloadStream", mock.Anything, "test-container", "test/read.txt", mock.Anything).Return(resp, nil)

	reader, meta, err := s.provider.ReadObjectContent(s.ctx, "test/read.txt")
	s.NoError(err)
	s.NotNil(reader)

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	s.Equal(content, buf.String())
	s.Equal("text/plain", meta["Content-Type"])
}

func (s *AzureTestSuite) TestDelete() {
	s.mockClient.On("DeleteBlob", mock.Anything, "test-container", "test/delete.txt", mock.Anything).Return(azblob.DeleteBlobResponse{}, nil)

	res, err := s.provider.Delete(s.ctx, "text/plain", []string{"test/delete.txt"})
	s.NoError(err)
	s.True(res)
}

func (s *AzureTestSuite) TestDeleteWithError() {
	s.mockClient.On("DeleteBlob", mock.Anything, "test-container", "test/fail.txt", mock.Anything).Return(azblob.DeleteBlobResponse{}, errors.New("delete failed"))

	res, err := s.provider.Delete(s.ctx, "text/plain", []string{"test/fail.txt"})
	s.Error(err)
	s.False(res)
}

func (s *AzureTestSuite) TestIsObjectExistsTrue() {
	s.mockClient.On("GetProperties", mock.Anything, "test-container", "exist.txt", mock.Anything).Return(blob.GetPropertiesResponse{}, nil)

	exists, err := s.provider.IsObjectExists(s.ctx, "exist.txt")
	s.NoError(err)
	s.True(exists)
}

func (s *AzureTestSuite) TestIsObjectExistsNotFound() {
	respErr := &azcore.ResponseError{StatusCode: 404}
	s.mockClient.On("GetProperties", mock.Anything, "test-container", "missing.txt", mock.Anything).Return(blob.GetPropertiesResponse{}, respErr)

	exists, err := s.provider.IsObjectExists(s.ctx, "missing.txt")
	s.NoError(err)
	s.False(exists)
}

func (s *AzureTestSuite) TestOverrideNotExists() {
	respErr := &azcore.ResponseError{StatusCode: 404}
	s.mockClient.On("GetProperties", mock.Anything, "test-container", "missing.txt", mock.Anything).Return(blob.GetPropertiesResponse{}, respErr)

	content := []byte("new content")
	_, err := s.provider.Override(s.ctx, e.TextPlain, "missing.txt", bytes.NewReader(content), int64(len(content)), nil)
	s.Error(err)
	s.Equal(e.ErrFetchOriginObject, err)
}

func (s *AzureTestSuite) TestGenReadPresignedURL() {
	_, err := s.provider.GenReadPresignedURL(s.ctx, "test.txt", 5*time.Minute)
	s.Equal(e.ErrNotImpl, err)
}

func (s *AzureTestSuite) TestPutPresignedURL() {
	_, err := s.provider.PutPresignedURL(s.ctx, "test.txt", e.JSON, 5*time.Minute, nil)
	s.Equal(e.ErrNotImpl, err)
}

func (s *AzureTestSuite) TestHealth() {
	status, err := s.provider.Health(s.ctx)
	s.NoError(err)
	s.Equal(e.Azure, status.Cloud)
}

func (s *AzureTestSuite) TestClose() {
	s.provider.Close() // should not panic
}

func (s *AzureTestSuite) TestNewAzure() {
	conf := &e.Config{
		CSP:    e.Azure,
		Bucket: "test",
	}
	result, err := NewAzure(s.ctx, conf)
	s.Nil(result)
	s.Equal(e.ErrNotImpl, err)
}
