package provider

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	e "github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

// MockMinioClient
type MockMinioClient struct {
	mock.Mock
}

func (m *MockMinioClient) PutObject(bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	args := m.Called(bucketName, objectName, reader, objectSize, opts)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMinioClient) GetObject(bucketName, objectName string, opts minio.GetObjectOptions) (io.ReadCloser, error) {
	args := m.Called(bucketName, objectName, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockMinioClient) StatObject(bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	args := m.Called(bucketName, objectName, opts)
	return args.Get(0).(minio.ObjectInfo), args.Error(1)
}

func (m *MockMinioClient) RemoveObjects(bucketName string, objectsCh <-chan string) <-chan minio.RemoveObjectError {
	args := m.Called(bucketName, objectsCh)
	return args.Get(0).(<-chan minio.RemoveObjectError)
}

func (m *MockMinioClient) PresignedGetObject(bucketName, objectName string, expires time.Duration, reqParams url.Values) (u *url.URL, err error) {
	args := m.Called(bucketName, objectName, expires, reqParams)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*url.URL), args.Error(1)
}

func (m *MockMinioClient) PresignedPutObject(bucketName, objectName string, expires time.Duration) (u *url.URL, err error) {
	args := m.Called(bucketName, objectName, expires)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*url.URL), args.Error(1)
}

func (m *MockMinioClient) ListObjects(bucketName, objectPrefix string, recursive bool, doneCh <-chan struct{}) <-chan minio.ObjectInfo {
	args := m.Called(bucketName, objectPrefix, recursive, doneCh)
	return args.Get(0).(<-chan minio.ObjectInfo)
}

func (m *MockMinioClient) ListBuckets() ([]minio.BucketInfo, error) {
	args := m.Called()
	return args.Get(0).([]minio.BucketInfo), args.Error(1)
}

type MinioTestSuite struct {
	suite.Suite
	mockClient *MockMinioClient
	provider   *minioImpl
	ctx        ctx.CTX
}

func TestMinioTestSuite(t *testing.T) {
	suite.Run(t, new(MinioTestSuite))
}

func (s *MinioTestSuite) SetupTest() {
	s.mockClient = new(MockMinioClient)
	s.ctx = ctx.Background()
	s.provider = &minioImpl{
		client:   s.mockClient,
		bucket:   "test-bucket",
		region:   "us-east-1",
		endpoint: "localhost:9000",
	}
}

func (s *MinioTestSuite) TestUploadSimple() {
	content := []byte("simple content")
	reader := bytes.NewReader(content)

	// StatObject is called to check existence first
	s.mockClient.On("StatObject", "test-bucket", "test/file.txt", mock.Anything).Return(minio.ObjectInfo{}, minio.ErrorResponse{Code: "NoSuchKey"}).Once()

	// PutObject
	s.mockClient.On("PutObject", "test-bucket", "test/file.txt", mock.Anything, int64(len(content)), mock.Anything).Return(int64(len(content)), nil).Run(func(args mock.Arguments) {
		r := args.Get(2).(io.Reader)
		buf, _ := io.ReadAll(r)
		s.Equal(string(content), string(buf))

		// Verify opts contain ContentType
		opts := args.Get(4).(minio.PutObjectOptions)
		s.Equal("text/plain", opts.ContentType)
	})

	// GenReadPresignedURL -> PresignedGetObject
	u, _ := url.Parse("http://localhost:9000/presigned")
	s.mockClient.On("PresignedGetObject", "test-bucket", "test/file.txt", mock.Anything, mock.Anything).Return(u, nil)

	urlStr, preUrl, err := s.provider.Upload(s.ctx, e.TextPlain, "test/file.txt", reader, int64(len(content)), nil)

	s.NoError(err)
	s.Contains(urlStr, "test-bucket/test/file.txt")
	s.Equal("http://localhost:9000/presigned", preUrl)
	s.mockClient.AssertExpectations(s.T())
}

func (s *MinioTestSuite) TestReadObjectContent() {
	bodyContent := "file content"
	bodyRC := io.NopCloser(bytes.NewReader([]byte(bodyContent)))

	// Mock StatObject
	s.mockClient.On("StatObject", "test-bucket", "test/read.txt", mock.Anything).Return(minio.ObjectInfo{
		Key:         "test/read.txt",
		Size:        int64(len(bodyContent)),
		ContentType: "text/plain",
	}, nil)

	// Mock GetObject
	s.mockClient.On("GetObject", "test-bucket", "test/read.txt", mock.Anything).Return(bodyRC, nil)

	reader, meta, err := s.provider.ReadObjectContent(s.ctx, "https://localhost:9000/test-bucket/test/read.txt")
	s.NoError(err)
	s.NotNil(reader)

	// Read from reader
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	s.Equal(bodyContent, buf.String())
	s.Equal("text/plain", meta["Content-Type"])

	s.mockClient.AssertExpectations(s.T())
}

func (s *MinioTestSuite) TestDelete() {
	objPaths := []string{"https://localhost:9000/test-bucket/test1", "https://localhost:9000/test-bucket/test2"}

	errCh := make(chan minio.RemoveObjectError)
	close(errCh) // Empty channel means no errors

	s.mockClient.On("RemoveObjects", "test-bucket", mock.Anything).Return((<-chan minio.RemoveObjectError)(errCh)).Run(func(args mock.Arguments) {
		ch := args.Get(1).(<-chan string)
		go func() {
			// Drain input channel to simulate processing
			for range ch {
			}
		}()
	})

	success, err := s.provider.Delete(s.ctx, e.TextPlain, objPaths)
	s.True(success)
	s.NoError(err)
	s.mockClient.AssertExpectations(s.T())
}

func (s *MinioTestSuite) TestIsObjectExistsTrue() {
	s.mockClient.On("StatObject", "test-bucket", "existing.txt", mock.Anything).Return(minio.ObjectInfo{
		Key:  "existing.txt",
		Size: 100,
	}, nil)

	exists, err := s.provider.IsObjectExists(s.ctx, "https://localhost:9000/test-bucket/existing.txt")
	s.NoError(err)
	s.True(exists)
}

func (s *MinioTestSuite) TestIsObjectExistsNotFound() {
	s.mockClient.On("StatObject", "test-bucket", "missing.txt", mock.Anything).Return(minio.ObjectInfo{}, minio.ErrorResponse{Code: "NoSuchKey"})

	exists, err := s.provider.IsObjectExists(s.ctx, "https://localhost:9000/test-bucket/missing.txt")
	s.NoError(err)
	s.False(exists)
}

func (s *MinioTestSuite) TestOverrideSuccess() {
	content := []byte("updated content")

	// StatObject for existence check — object exists
	s.mockClient.On("StatObject", "test-bucket", "test/override.txt", mock.Anything).Return(minio.ObjectInfo{
		Key:         "test/override.txt",
		Size:        50,
		ContentType: "text/plain",
	}, nil)

	// PutObject for upload
	s.mockClient.On("PutObject", "test-bucket", "test/override.txt", mock.Anything, int64(len(content)), mock.Anything).Return(int64(len(content)), nil)

	// GenReadPresignedURL
	u, _ := url.Parse("http://localhost:9000/presigned-override")
	s.mockClient.On("PresignedGetObject", "test-bucket", "test/override.txt", mock.Anything, mock.Anything).Return(u, nil)

	objURL, err := s.provider.Override(s.ctx, e.TextPlain, "https://localhost:9000/test-bucket/test/override.txt", bytes.NewReader(content), int64(len(content)), nil)
	s.NoError(err)
	s.Contains(objURL, "test-bucket/test/override.txt")
}

func (s *MinioTestSuite) TestOverrideNotExists() {
	content := []byte("content")

	s.mockClient.On("StatObject", "test-bucket", "missing.txt", mock.Anything).Return(minio.ObjectInfo{}, minio.ErrorResponse{Code: "NoSuchKey"})

	_, err := s.provider.Override(s.ctx, e.TextPlain, "https://localhost:9000/test-bucket/missing.txt", bytes.NewReader(content), int64(len(content)), nil)
	s.Error(err)
	s.Equal(e.ErrFetchOriginObjFromS3ByGivenObjPath, err)
}

func (s *MinioTestSuite) TestGenReadPresignedURL() {
	u, _ := url.Parse("http://localhost:9000/presigned-get")
	s.mockClient.On("PresignedGetObject", "test-bucket", "test/read.txt", mock.Anything, mock.Anything).Return(u, nil)

	readURL, err := s.provider.GenReadPresignedURL(s.ctx, "https://localhost:9000/test-bucket/test/read.txt", 5*time.Minute)
	s.NoError(err)
	s.Equal("http://localhost:9000/presigned-get", readURL)
}

func (s *MinioTestSuite) TestPutPresignedURL() {
	// StatObject — object does not exist
	s.mockClient.On("StatObject", "test-bucket", "test/put.txt", mock.Anything).Return(minio.ObjectInfo{}, minio.ErrorResponse{Code: "NoSuchKey"})

	u, _ := url.Parse("http://localhost:9000/presigned-put")
	s.mockClient.On("PresignedPutObject", "test-bucket", "test/put.txt", mock.Anything).Return(u, nil)

	putURL, err := s.provider.PutPresignedURL(s.ctx, "https://localhost:9000/test-bucket/test/put.txt", e.JSON, 5*time.Minute, nil)
	s.NoError(err)
	s.Equal("http://localhost:9000/presigned-put", putURL)
}

func (s *MinioTestSuite) TestHealth() {
	s.mockClient.On("ListBuckets").Return([]minio.BucketInfo{
		{Name: "test-bucket"},
	}, nil)

	status, err := s.provider.Health(s.ctx)
	s.NoError(err)
	s.Equal(e.Minio, status.Cloud)
}

func (s *MinioTestSuite) TestHealthFailed() {
	s.mockClient.On("ListBuckets").Return([]minio.BucketInfo{}, errors.New("connection refused"))

	status, err := s.provider.Health(s.ctx)
	s.Error(err)
	s.Equal(e.Minio, status.Cloud)
}

func (s *MinioTestSuite) TestClose() {
	s.provider.Close() // should not panic
}
