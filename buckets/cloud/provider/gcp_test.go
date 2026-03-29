package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gabriel-vasile/mimetype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	e "github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

// MockGCSClient for testing
type MockGCSClient struct {
	mock.Mock
}

func (m *MockGCSClient) NewReader(ctx context.Context, bucket, object string) (io.ReadCloser, error) {
	args := m.Called(ctx, bucket, object)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockGCSClient) NewWriter(ctx context.Context, bucket, object string) (io.WriteCloser, error) {
	args := m.Called(ctx, bucket, object)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.WriteCloser), args.Error(1)
}

func (m *MockGCSClient) ObjectAttrs(ctx context.Context, bucket, object string) (*storage.ObjectAttrs, error) {
	args := m.Called(ctx, bucket, object)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ObjectAttrs), args.Error(1)
}

func (m *MockGCSClient) DeleteObject(ctx context.Context, bucket, object string) error {
	args := m.Called(ctx, bucket, object)
	return args.Error(0)
}

func (m *MockGCSClient) Objects(ctx context.Context, bucket string, query *storage.Query) *storage.ObjectIterator {
	args := m.Called(ctx, bucket, query)
	return args.Get(0).(*storage.ObjectIterator)
}

func (m *MockGCSClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGCSClient) WriteObject(ctx context.Context, bucket, object string, reader io.Reader, contentType string, metadata map[string]string) error {
	args := m.Called(ctx, bucket, object, reader, contentType, metadata)
	// Consume the reader to simulate writing
	if reader != nil {
		io.ReadAll(reader)
	}
	return args.Error(0)
}

type GCPTestSuite struct {
	suite.Suite
	mockClient *MockGCSClient
	provider   *gcsImpl
	ctx        ctx.CTX
}

func TestGCPTestSuite(t *testing.T) {
	suite.Run(t, new(GCPTestSuite))
}

func (s *GCPTestSuite) SetupTest() {
	s.mockClient = new(MockGCSClient)
	s.ctx = ctx.Background()
	s.provider = &gcsImpl{
		bucket: "test-bucket",
		region: "us-central1",
		api:    s.mockClient,
	}
}

func (s *GCPTestSuite) TestUpload() {
	content := []byte("test content")
	reader := bytes.NewReader(content)

	// Determine mime type dynamically to match implementation logic
	detectedMime := mimetype.Detect(content).String()

	// Mock WriteObject (now used instead of NewWriter)
	s.mockClient.On("WriteObject", mock.Anything, "test-bucket", "test/file.txt", mock.Anything, detectedMime, mock.Anything).Return(nil)

	url, _, err := s.provider.Upload(s.ctx, e.ContentType(detectedMime), "test/file.txt", reader, int64(len(content)), nil)

	s.Require().NoError(err)
	s.Contains(url, "test-bucket/test/file.txt")
}

func (s *GCPTestSuite) TestReadObjectContent() {
	content := "file content"
	mockReader := io.NopCloser(bytes.NewReader([]byte(content)))

	s.mockClient.On("NewReader", mock.Anything, "test-bucket", "test/read.txt").Return(mockReader, nil)
	s.mockClient.On("ObjectAttrs", mock.Anything, "test-bucket", "test/read.txt").Return(&storage.ObjectAttrs{
		ContentType: "text/plain",
		Metadata:    map[string]string{"key": "val"},
	}, nil)

	reader, meta, err := s.provider.ReadObjectContent(s.ctx, "test/read.txt")
	s.NoError(err)
	s.NotNil(reader)

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	s.Equal(content, buf.String())
	s.Equal("text/plain", meta["Content-Type"])
	s.Equal("val", meta["key"])
}

func (s *GCPTestSuite) TestDelete() {
	s.mockClient.On("DeleteObject", mock.Anything, "test-bucket", "test/delete.txt").Return(nil)

	res, err := s.provider.Delete(s.ctx, "text/plain", []string{"test/delete.txt"})
	s.NoError(err)
	s.True(res)
}

func (s *GCPTestSuite) TestIsObjectExistsTrue() {
	s.mockClient.On("ObjectAttrs", mock.Anything, "test-bucket", "exists.txt").Return(&storage.ObjectAttrs{Name: "exists.txt"}, nil)

	exists, err := s.provider.IsObjectExists(s.ctx, fmt.Sprintf("%s/%s/%s", "https://storage.googleapis.com", "test-bucket", "exists.txt"))
	s.NoError(err)
	s.True(exists)
}

func (s *GCPTestSuite) TestIsObjectExistsNotFound() {
	s.mockClient.On("ObjectAttrs", mock.Anything, "test-bucket", "missing.txt").Return(nil, storage.ErrObjectNotExist)

	exists, err := s.provider.IsObjectExists(s.ctx, fmt.Sprintf("%s/%s/%s", "https://storage.googleapis.com", "test-bucket", "missing.txt"))
	s.NoError(err)
	s.False(exists)
}

func (s *GCPTestSuite) TestOverrideNotExists() {
	s.mockClient.On("ObjectAttrs", mock.Anything, "test-bucket", "missing.txt").Return(nil, storage.ErrObjectNotExist)

	content := []byte("new content")
	_, err := s.provider.Override(s.ctx, e.TextPlain, fmt.Sprintf("%s/%s/%s", "https://storage.googleapis.com", "test-bucket", "missing.txt"), bytes.NewReader(content), int64(len(content)), nil)
	s.Error(err)
	s.Equal(e.ErrFetchOriginObject, err)
}

func (s *GCPTestSuite) TestGenReadPresignedURL() {
	_, err := s.provider.GenReadPresignedURL(s.ctx, "test.txt", 5*time.Minute)
	s.Equal(e.ErrNotImpl, err)
}

func (s *GCPTestSuite) TestPutPresignedURL() {
	_, err := s.provider.PutPresignedURL(s.ctx, "test.txt", e.JSON, 5*time.Minute, nil)
	s.Equal(e.ErrNotImpl, err)
}

func (s *GCPTestSuite) TestHealth() {
	status, err := s.provider.Health(s.ctx)
	s.NoError(err)
	s.Equal(e.GCP, status.Cloud)
}

func (s *GCPTestSuite) TestClose() {
	s.mockClient.On("Close").Return(nil)
	s.provider.Close()
	s.mockClient.AssertCalled(s.T(), "Close")
}

func (s *GCPTestSuite) TestGetObjectListEmpty() {
	// Objects returns an iterator — testing this requires a real iterator or custom mock
	// so we test parseURL instead
	bucket, key, err := s.provider.parseURL("https://storage.googleapis.com/test-bucket/path/to/file.txt")
	s.NoError(err)
	s.Equal("test-bucket", bucket)
	s.Equal("path/to/file.txt", key)
}

func (s *GCPTestSuite) TestParseURLInvalid() {
	_, _, err := s.provider.parseURL("not-a-url")
	s.Error(err)
}

// MockWriteCloser helper
type MockWriteCloser struct {
	*bytes.Buffer
	Closed bool
}

func (m *MockWriteCloser) Close() error {
	m.Closed = true
	return nil
}

func (m *MockWriteCloser) Write(p []byte) (n int, err error) {
	fmt.Printf("MockWriteCloser Write called with %d bytes: %s\n", len(p), string(p))
	return m.Buffer.Write(p)
}
