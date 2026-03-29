package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	cpIfc "github.com/hsuanshao/golibs/buckets/cloud/provider/interface"
	e "github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

/**
s3 & minio bucket unit test
*/

var (
	mockCTX           = ctx.Background()
	defaultRegion     = "ap-northeast-1"
	defaultBucket     = "hsuanshao-bucket"
	defaultS3api      = new(MockS3Client)
	defaultPresignApi = new(MockS3PresignClient)
)

// MockS3Client mocks S3ClientAPI
type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *awsS3.ListObjectsV2Input, optFns ...func(*awsS3.Options)) (*awsS3.ListObjectsV2Output, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*awsS3.ListObjectsV2Output), args.Error(1)
}

func (m *MockS3Client) GetObject(ctx context.Context, params *awsS3.GetObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.GetObjectOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*awsS3.GetObjectOutput), args.Error(1)
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *awsS3.HeadObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.HeadObjectOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*awsS3.HeadObjectOutput), args.Error(1)
}

func (m *MockS3Client) PutObject(ctx context.Context, params *awsS3.PutObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.PutObjectOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*awsS3.PutObjectOutput), args.Error(1)
}

func (m *MockS3Client) DeleteObjects(ctx context.Context, params *awsS3.DeleteObjectsInput, optFns ...func(*awsS3.Options)) (*awsS3.DeleteObjectsOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*awsS3.DeleteObjectsOutput), args.Error(1)
}

func (m *MockS3Client) GetObjectAttributes(ctx context.Context, params *awsS3.GetObjectAttributesInput, optFns ...func(*awsS3.Options)) (*awsS3.GetObjectAttributesOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*awsS3.GetObjectAttributesOutput), args.Error(1)
}

// MockS3PresignClient mocks S3PresignClientAPI
type MockS3PresignClient struct {
	mock.Mock
}

func (m *MockS3PresignClient) PresignGetObject(ctx context.Context, params *awsS3.GetObjectInput, optFns ...func(*awsS3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4.PresignedHTTPRequest), args.Error(1)
}

func (m *MockS3PresignClient) PresignPutObject(ctx context.Context, params *awsS3.PutObjectInput, optFns ...func(*awsS3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4.PresignedHTTPRequest), args.Error(1)
}

// MockConn for net.Conn
type MockConn struct {
	mock.Mock
}

func (m *MockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *MockConn) Write(b []byte) (n int, err error)  { return 0, nil }
func (m *MockConn) Close() error                       { return m.Called().Error(0) }
func (m *MockConn) LocalAddr() net.Addr                { return nil }
func (m *MockConn) RemoteAddr() net.Addr               { return nil }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

type s3TestSuite struct {
	suite.Suite
	CSP cpIfc.ObjectServiceProvider
}

func TestS3Suite(t *testing.T) {
	ctx.SetDebugLevel()
	suite.Run(t, new(s3TestSuite))
}

func (s *s3TestSuite) SetupTest() {
	defaultS3api = new(MockS3Client)
	defaultPresignApi = new(MockS3PresignClient)
	s.CSP = &s3impl{
		s3Srv:         defaultS3api,
		presignClient: defaultPresignApi,
		region:        defaultRegion,
		bucket:        defaultBucket,
	}
}

// TearDownSuite please applied to in the end of any test case to reset
// test suite back to default status
func (s *s3TestSuite) TearDownSuite() {
	mockCTX = ctx.Background()
	defaultS3api = new(MockS3Client)
	defaultPresignApi = new(MockS3PresignClient)
	s.CSP = &s3impl{
		s3Srv:         defaultS3api,
		presignClient: defaultPresignApi,
		region:        defaultRegion,
		bucket:        defaultBucket,
	}
	dialTimeout = net.DialTimeout
}

// Test GetObjectList
func (s *s3TestSuite) TestGetObjectList() {
	testcase := []struct {
		Case     string
		MockFunc func()
		Prefix   string
		Delim    string
		ExpRes   []string
		ExpErr   error
	}{
		{
			Case: "normal case, has prefix, without delim",
			MockFunc: func() {
				defaultS3api.On("ListObjectsV2", mock.Anything, &awsS3.ListObjectsV2Input{
					Bucket: aws.String(defaultBucket),
					Prefix: aws.String("config"),
				}, mock.Anything).Return(&awsS3.ListObjectsV2Output{
					Contents: []types.Object{
						{
							Key: aws.String("/config/indexer_setup.json"),
						},
						{
							Key: aws.String("/config/buckets.json"),
						},
					},
				}, nil).Once()
			},
			Prefix: "config",
			Delim:  "",
			ExpRes: []string{
				"https://hsuanshao-bucket.s3.amazonaws.com/config/indexer_setup.json",
				"https://hsuanshao-bucket.s3.amazonaws.com/config/buckets.json",
			},
			ExpErr: nil,
		},
		{
			Case: "normal case, has prefix, exclusive indexer",
			MockFunc: func() {
				defaultS3api.On("ListObjectsV2", mock.Anything, &awsS3.ListObjectsV2Input{
					Bucket:    aws.String(defaultBucket),
					Prefix:    aws.String("config"),
					Delimiter: aws.String("indexer"),
				}, mock.Anything).Return(&awsS3.ListObjectsV2Output{
					Contents: []types.Object{
						{
							Key: aws.String("/config/buckets.json"),
						},
					},
				}, nil).Once()
			},
			Prefix: "config",
			Delim:  "indexer",
			ExpRes: []string{
				"https://hsuanshao-bucket.s3.amazonaws.com/config/buckets.json",
			},
			ExpErr: nil,
		},
		{
			Case: "use a new bucket client",
			MockFunc: func() {
				s.CSP = &s3impl{
					s3Srv:  defaultS3api,
					region: "ap-south-1",
					bucket: "hsuanshao.ag",
				}

				defaultS3api.On("ListObjectsV2", mock.Anything, &awsS3.ListObjectsV2Input{
					Bucket: aws.String("hsuanshao.ag"),
					Prefix: aws.String("config"),
				}, mock.Anything).Return(&awsS3.ListObjectsV2Output{
					Contents: []types.Object{
						{
							Key: aws.String("/config/indexer_setup.json"),
						},
						{
							Key: aws.String("/config/buckets.json"),
						},
					},
				}, nil).Once()
			},
			Prefix: "config",
			Delim:  "",
			ExpRes: []string{
				"https://s3-ap-south-1.amazonaws.com/hsuanshao.ag/config/indexer_setup.json",
				"https://s3-ap-south-1.amazonaws.com/hsuanshao.ag/config/buckets.json",
			},
			ExpErr: nil,
		},
		{
			Case: "use a new bucket client, but given wrong bucket",
			MockFunc: func() {
				s.CSP = &s3impl{
					s3Srv:  defaultS3api,
					region: "ap-south-1",
					bucket: defaultBucket,
				}

				defaultS3api.On("ListObjectsV2", mock.Anything, &awsS3.ListObjectsV2Input{
					Bucket: aws.String(defaultBucket),
					Prefix: aws.String("config"),
				}, mock.Anything).Return(nil, fmt.Errorf("NoSuchBucket, %v", defaultBucket)).Once()
			},
			Prefix: "config",
			Delim:  "",
			ExpRes: nil,
			ExpErr: e.ErrFetchObjList,
		},
	}

	for idx, c := range testcase {
		mockCTX = ctx.WithValue(mockCTX, "caes no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		// run mock func
		c.MockFunc()

		objURLs, err := s.CSP.GetObjectList(mockCTX, c.Prefix, c.Delim)
		s.Equal(c.ExpErr, err, fmt.Sprintf("Case[%v]%v: error not as expected", idx, c.Case))

		s.Equal(len(c.ExpRes), len(objURLs), fmt.Sprintf("Case[%v]%v: compare return urls slice length", idx, c.Case))

		for urlIDX, url := range c.ExpRes {
			s.Equal(url, objURLs[urlIDX], fmt.Sprintf("[C]return urls on %v, get not match url, expected URL is %v, but get %v", urlIDX, url, objURLs[urlIDX]))
		}

		// Note: tear down suite to help reset test suite return to original test setup
		s.TearDownSuite()
	}
}

func (s *s3TestSuite) TestReadObjectContent() {
	testcase := []struct {
		Case        string
		MockFunc    func()
		ObjURL      string
		ExpRes      []byte
		ExpMetadata map[string]string
		ExpErr      error
	}{
		{
			Case: "Normal case",
			MockFunc: func() {
				stringRead := strings.NewReader(`[{"cloud":"aws","purpose":"i18n","region":"ap-northeast-1","bucket":"hsuanshao.system.i18n"},{"cloud":"aws","purpose":"config","region":"ap-northeast-1","bucket":"hsuanshao.system.conf"}]`)
				stringRC := io.NopCloser(stringRead)
				defaultS3api.On("GetObject", mock.Anything, &awsS3.GetObjectInput{Bucket: aws.String(defaultBucket), Key: aws.String("/config/bucket.json")}, mock.Anything).Return(&awsS3.GetObjectOutput{
					ContentType: aws.String("application/json"),
					Metadata: map[string]string{
						"for-system": "indexer",
						"version":    "v1.0",
					},
					ContentLength: aws.Int64(175),
					Body:          stringRC,
				}, nil).Once()
			},
			ObjURL: "/config/bucket.json",
			ExpRes: []byte(`[{"cloud":"aws","purpose":"i18n","region":"ap-northeast-1","bucket":"hsuanshao.system.i18n"},{"cloud":"aws","purpose":"config","region":"ap-northeast-1","bucket":"hsuanshao.system.conf"}]`),
			ExpMetadata: map[string]string{
				"for-system": "indexer",
				"version":    "v1.0",
			},
			ExpErr: nil,
		},
		{
			Case: "Object not exits case",
			MockFunc: func() {

				defaultS3api.On("GetObject", mock.Anything, &awsS3.GetObjectInput{Bucket: aws.String(defaultBucket), Key: aws.String("/config/bucket-special.json")}, mock.Anything).Return(&awsS3.GetObjectOutput{
					ContentType:   nil,
					Metadata:      nil,
					ContentLength: aws.Int64(0),
					Body:          nil,
				}, fmt.Errorf("any error")).Once()
			},
			ObjURL:      "/config/bucket-special.json",
			ExpRes:      nil,
			ExpMetadata: nil,
			ExpErr:      e.ErrGetObjFromS3,
		},
	}

	for idx, c := range testcase {
		mockCTX = ctx.WithValue(mockCTX, "caes no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		respReader, metadata, err := s.CSP.ReadObjectContent(mockCTX, c.ObjURL)

		var respBytes []byte
		if respReader != nil {
			respBytes, _ = io.ReadAll(respReader)
			respReader.Close()
		}

		if !s.Equal(c.ExpRes, respBytes) {
			mockCTX.Error("expected result not match")
		}

		if !s.Equal(len(c.ExpMetadata), len(metadata)) {
			mockCTX.Error("expected metadata of object header length not match")
		}

		for key, val := range c.ExpMetadata {
			if !s.Equal(val, metadata[key]) {
				mockCTX.WithFields(logrus.Fields{"key": key, "metadata val": metadata[key]}).Error("the key in metadata return is not match expected result")
			}
		}

		s.Equal(c.ExpErr, err, fmt.Sprintf("Case[%v]%v, expected error not match", idx, c.Case))
		s.TearDownSuite()
	}
}

func (s *s3TestSuite) TestIsObjectExists() {
	testcase := []struct {
		Case     string
		MockFunc func()
		ObjURL   string
		ExpRes   bool
		ExpErr   error
	}{
		{
			Case: "exists case",
			MockFunc: func() {
				mockModifiedTimeStr := "2022-11-01T12:00:00-00:00"
				mockTimeObj, _ := time.Parse(time.RFC3339, mockModifiedTimeStr)
				// Replaced GetObjectAttributes with HeadObject
				defaultS3api.On("HeadObject", mock.Anything, &awsS3.HeadObjectInput{
					Bucket: aws.String(defaultBucket),
					Key:    aws.String("config/buckets.json"),
				}, mock.Anything).Return(&awsS3.HeadObjectOutput{
					LastModified: aws.Time(mockTimeObj),
					VersionId:    nil,
				}, nil).Once()
			},
			ObjURL: "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/config/buckets.json",
			ExpRes: true,
			ExpErr: nil,
		},
		{
			Case:     "has no permission access (region issue)",
			MockFunc: func() {},
			ObjURL:   "https://s3-ap-south-1.amazonaws.com/hsuanshao.ag/config/buckets.json",
			ExpRes:   false,
			ExpErr:   e.ErrWithoutPermissionToAccess,
		},
		{
			Case:     "has no permission access (bucket issue)",
			MockFunc: func() {},
			ObjURL:   "https://s3-ap-northeast-1.amazonaws.com/hsuanshao.ag/config/buckets.json",
			ExpRes:   false,
			ExpErr:   e.ErrWithoutPermissionToAccess,
		},
	}

	for idx, c := range testcase {
		mockCTX = ctx.WithValue(mockCTX, "caes no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		objExsists, err := s.CSP.IsObjectExists(mockCTX, c.ObjURL)
		if !s.Equal(c.ExpRes, objExsists) {
			mockCTX.Error("expected result not match")
		}

		if !s.Equal(c.ExpErr, err) {
			mockCTX.Error("expected error not match")
		}

		s.TearDownSuite()
	}
}

func (s *s3TestSuite) TestGenReadPresignedURL() {
	testcases := []struct {
		Case     string
		MockFunc func()
		ObjURL   string
		Duration time.Duration
		ExpURL   string
		ExpErr   error
	}{
		{
			Case:     "Region is incorrect",
			MockFunc: func() {},
			ObjURL:   "https://s3-ap-east-1.amazonaws.com/hsuanshao.ag/indexer/config/decoder_conf.yml",
			Duration: 10 * time.Minute,
			ExpURL:   "",
			ExpErr:   e.ErrWithoutPermissionToAccess,
		},
		{
			Case:     "Bucket is incorrect",
			MockFunc: func() {},
			ObjURL:   "https://s3-ap-northeast-1.amazonaws.com/hsuanshao.ag/indexer/config/decoder_conf.yml",
			Duration: 10 * time.Minute,
			ExpURL:   "",
			ExpErr:   e.ErrWithoutPermissionToAccess,
		},
		{
			Case: "Normal case",
			MockFunc: func() {
				defaultPresignApi.On("PresignGetObject", mock.Anything, &awsS3.GetObjectInput{
					Bucket: aws.String("hsuanshao-bucket"),
					Key:    aws.String("indexer/config/decoder_conf.yml"),
				}, mock.Anything).Return(&v4.PresignedHTTPRequest{
					URL: "https://s3-ap-east-1.amazonaws.com/hsuanshao.ag/indexer/config/decoder_conf.yml?X-Amz-Algorithm=AW54-HMAC-SHA256&X-Amz-Date=20221130T150956&X-Amz-SignedHeaders=host&X-Amz-Expires=900&X-Amz-Credential=AKIAYIDZSD2%202211%2Fap-northeast-1%2Fs3%2Faws4_request&aws4_request&X-amz-Signature=062ddg453248sdf3298df98u3984dsasd34sd",
				}, nil).Once()
			},
			ObjURL:   "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/indexer/config/decoder_conf.yml",
			Duration: 10 * time.Minute,
			ExpURL:   "https://s3-ap-east-1.amazonaws.com/hsuanshao.ag/indexer/config/decoder_conf.yml?X-Amz-Algorithm=AW54-HMAC-SHA256&X-Amz-Date=20221130T150956&X-Amz-SignedHeaders=host&X-Amz-Expires=900&X-Amz-Credential=AKIAYIDZSD2%202211%2Fap-northeast-1%2Fs3%2Faws4_request&aws4_request&X-amz-Signature=062ddg453248sdf3298df98u3984dsasd34sd",
			ExpErr:   nil,
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		readURL, err := s.CSP.GenReadPresignedURL(mockCTX, c.ObjURL, c.Duration)

		s.Equal(c.ExpURL, readURL, fmt.Sprintf("Case[%v]%v: check output read presigned url but get unexpected url", idx, c.Case))
		s.Equal(c.ExpErr, err, fmt.Sprintf("Case[%v]%v: check error get inconsist result", idx, c.Case))

		s.TearDownSuite()
	}
}

func (s *s3TestSuite) TestPutPresignedURL() {
	testcases := []struct {
		Case     string
		MockFunc func()
		ObjURL   string
		Mime     e.ContentType
		Duration time.Duration
		Metadata map[string]string
		ExpURL   string
		ExpErr   error
	}{
		{
			Case:     "Region is incorrect",
			MockFunc: func() {},
			ObjURL:   "https://s3-ap-east-1.amazonaws.com/hsuanshao.ag/indexer/config/decoder_conf.yml",
			Mime:     e.JSON,
			Duration: 10 * time.Minute,
			ExpURL:   "",
			ExpErr:   e.ErrWithoutPermissionToAccess,
		},
		{
			Case: "Normal case, object not exists",
			MockFunc: func() {
				// Mock IsObjectExists -> HeadObject returns error
				defaultS3api.On("HeadObject", mock.Anything, &awsS3.HeadObjectInput{
					Bucket: aws.String(defaultBucket),
					Key:    aws.String("indexer/config/decoder_conf.yml"),
				}, mock.Anything).Return(nil, errors.New("NotFound")).Once()

				defaultPresignApi.On("PresignPutObject", mock.Anything, &awsS3.PutObjectInput{
					Bucket:      aws.String(defaultBucket),
					Key:         aws.String("indexer/config/decoder_conf.yml"),
					ContentType: aws.String("application/json"),
				}, mock.Anything).Return(&v4.PresignedHTTPRequest{
					URL: "https://example.com/presigned-put",
				}, nil).Once()
			},
			ObjURL:   "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/indexer/config/decoder_conf.yml",
			Mime:     e.JSON,
			Duration: 10 * time.Minute,
			ExpURL:   "https://example.com/presigned-put",
			ExpErr:   nil,
		},
		{
			Case: "Object already exists",
			MockFunc: func() {
				// Mock IsObjectExists -> HeadObject returns success
				defaultS3api.On("HeadObject", mock.Anything, &awsS3.HeadObjectInput{
					Bucket: aws.String(defaultBucket),
					Key:    aws.String("indexer/config/decoder_conf.yml"),
				}, mock.Anything).Return(&awsS3.HeadObjectOutput{}, nil).Once()
			},
			ObjURL:   "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/indexer/config/decoder_conf.yml",
			Mime:     e.JSON,
			Duration: 10 * time.Minute,
			ExpURL:   "",
			ExpErr:   e.ErrObjectPathHasItem,
		},
		{
			Case: "PresignPutObject failed",
			MockFunc: func() {
				// Mock IsObjectExists -> HeadObject returns error
				defaultS3api.On("HeadObject", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("NotFound")).Once()

				defaultPresignApi.On("PresignPutObject", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("PresignError")).Once()
			},
			ObjURL:   "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/indexer/config/decoder_conf.yml",
			Mime:     e.JSON,
			Duration: 10 * time.Minute,
			ExpURL:   "",
			ExpErr:   e.ErrGenPutPresignedURL,
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		putObjURL, err := s.CSP.PutPresignedURL(mockCTX, c.ObjURL, c.Mime, c.Duration, c.Metadata)

		s.Equal(c.ExpURL, putObjURL, fmt.Sprintf("Case[%v]%v: check output put presigned url but get unexpected url", idx, c.Case))
		s.Equal(c.ExpErr, err, fmt.Sprintf("Case[%v]%v: check error get inconsist result", idx, c.Case))

		s.TearDownSuite()
	}
}

func (s *s3TestSuite) TestUpload() {
	objBytes := []byte(`{"foo":"bar"}`)
	testcases := []struct {
		Case                string
		MockFunc            func()
		Mime                e.ContentType
		ObjURL              string
		ObjByte             []byte
		ObjMetadata         map[string]string
		ExpURL              string
		ExpReadPresignedURL string
		ExpErr              error
	}{
		{
			Case: "Normal case",
			MockFunc: func() {
				// IsObjectExists -> returns NotFound
				defaultS3api.On("HeadObject", mock.Anything, &awsS3.HeadObjectInput{
					Bucket: aws.String(defaultBucket),
					Key:    aws.String("data/foo.json"),
				}, mock.Anything).Return(nil, errors.New("NotFound")).Once()

				// PutObject
				defaultS3api.On("PutObject", mock.Anything, mock.MatchedBy(func(input *awsS3.PutObjectInput) bool {
					return *input.Bucket == defaultBucket && *input.Key == "data/foo.json"
				}), mock.Anything).Return(&awsS3.PutObjectOutput{
					VersionId: aws.String("v1"),
				}, nil).Once()

				// GenReadPresignedURL -> PresignGetObject
				defaultPresignApi.On("PresignGetObject", mock.Anything, &awsS3.GetObjectInput{
					Bucket: aws.String(defaultBucket),
					Key:    aws.String("data/foo.json"),
				}, mock.Anything).Return(&v4.PresignedHTTPRequest{
					URL: "https://example.com/presigned-read",
				}, nil).Once()
			},
			Mime:    e.JSON,
			ObjURL:  "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/data/foo.json",
			ObjByte: objBytes,
			ObjMetadata: map[string]string{
				"author": "tester",
			},
			ExpURL:              "https://hsuanshao-bucket.s3.amazonaws.com/data/foo.json",
			ExpReadPresignedURL: "https://example.com/presigned-read",
			ExpErr:              nil,
		},
		{
			Case: "Object already exists",
			MockFunc: func() {
				// IsObjectExists -> returns success
				defaultS3api.On("HeadObject", mock.Anything, &awsS3.HeadObjectInput{
					Bucket: aws.String(defaultBucket),
					Key:    aws.String("data/foo.json"),
				}, mock.Anything).Return(&awsS3.HeadObjectOutput{}, nil).Once()
			},
			Mime:    e.JSON,
			ObjURL:  "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/data/foo.json",
			ObjByte: objBytes,
			ExpURL:  "",
			ExpErr:  e.ErrObjectPathHasItem,
		},
		{
			Case: "Content type mismatch",
			MockFunc: func() {
				// IsObjectExists -> returns NotFound
				defaultS3api.On("HeadObject", mock.Anything, &awsS3.HeadObjectInput{
					Bucket: aws.String(defaultBucket),
					Key:    aws.String("data/foo.json"),
				}, mock.Anything).Return(nil, errors.New("NotFound")).Once()
			},
			Mime:    e.PNG, // mismatch
			ObjURL:  "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/data/foo.json",
			ObjByte: objBytes, // JSON content
			ExpURL:  "",
			ExpErr:  e.ErrUploadNotMatchContentType,
		},
		{
			Case: "PutObject failed",
			MockFunc: func() {
				// IsObjectExists -> returns NotFound
				defaultS3api.On("HeadObject", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("NotFound")).Once()

				// PutObject -> fails
				defaultS3api.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("PutError")).Once()
			},
			Mime:    e.JSON,
			ObjURL:  "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/data/foo.json",
			ObjByte: objBytes,
			ExpURL:  "",
			ExpErr:  e.ErrUploadObjToS3,
		},
		{
			Case: "GenReadPresignedURL failed after upload",
			MockFunc: func() {
				// IsObjectExists -> returns NotFound
				defaultS3api.On("HeadObject", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("NotFound")).Once()

				// PutObject -> success
				defaultS3api.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(&awsS3.PutObjectOutput{}, nil).Once()

				// GenReadPresignedURL -> fails
				defaultPresignApi.On("PresignGetObject", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("PresignError")).Once()
			},
			Mime:    e.JSON,
			ObjURL:  "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/data/foo.json",
			ObjByte: objBytes,
			ExpURL:  "https://hsuanshao-bucket.s3.amazonaws.com/data/foo.json",
			ExpErr:  nil, // It returns nil error, just empty readPresignedURL
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		objURL, readPresignedURL, err := s.CSP.Upload(mockCTX, c.Mime, c.ObjURL, bytes.NewReader(c.ObjByte), int64(len(c.ObjByte)), c.ObjMetadata)

		s.Equal(c.ExpURL, objURL, fmt.Sprintf("Case[%v]%v: check output obj url but get unexpected url", idx, c.Case))
		s.Equal(c.ExpReadPresignedURL, readPresignedURL, fmt.Sprintf("Case[%v]%v: check output obj read presigned url but get unexpected url", idx, c.Case))
		s.Equal(c.ExpErr, err, fmt.Sprintf("Case[%v]%v: check error get inconsist result", idx, c.Case))

		s.TearDownSuite()
	}
}

func (s *s3TestSuite) TestOverride() {
	objBytes := []byte(`{"foo":"bar"}`)
	testcases := []struct {
		Case        string
		MockFunc    func()
		Mime        e.ContentType
		ObjURL      string
		ObjByte     []byte
		ObjMetadata map[string]string
		ExpURL      string
		ExpErr      error
	}{
		{
			Case: "Normal override",
			MockFunc: func() {
				// HeadObject for check existence
				defaultS3api.On("HeadObject", mock.Anything, &awsS3.HeadObjectInput{
					Bucket: aws.String(defaultBucket),
					Key:    aws.String("data/foo.json"),
				}, mock.Anything).Return(&awsS3.HeadObjectOutput{
					LastModified: aws.Time(time.Now()),
					ContentType:  aws.String("application/json"),
				}, nil).Once()

				// PutObject
				defaultS3api.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(&awsS3.PutObjectOutput{
					VersionId: aws.String("v2"),
				}, nil).Once()

				// GenReadPresignedURL (called inside uploadCore)
				defaultPresignApi.On("PresignGetObject", mock.Anything, mock.Anything, mock.Anything).Return(&v4.PresignedHTTPRequest{
					URL: "https://example.com/presigned-read-v2",
				}, nil).Once()
			},
			Mime:    e.JSON,
			ObjURL:  "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/data/foo.json",
			ObjByte: objBytes,
			ExpURL:  "https://hsuanshao-bucket.s3.amazonaws.com/data/foo.json",
			ExpErr:  nil,
		},
		{
			Case: "Original Object not found",
			MockFunc: func() {
				defaultS3api.On("HeadObject", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("NotFound")).Once()
			},
			Mime:    e.JSON,
			ObjURL:  "https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/data/foo.json",
			ObjByte: objBytes,
			ExpURL:  "",
			ExpErr:  e.ErrFetchOriginObjFromS3ByGivenObjPath,
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		objURL, err := s.CSP.Override(mockCTX, c.Mime, c.ObjURL, bytes.NewReader(c.ObjByte), int64(len(c.ObjByte)), c.ObjMetadata)

		s.Equal(c.ExpURL, objURL, fmt.Sprintf("Case[%v]%v: check output obj url but get unexpected url", idx, c.Case))

		s.Equal(c.ExpErr, err, fmt.Sprintf("Case[%v]%v: check error get inconsist result", idx, c.Case))

		s.TearDownSuite()
	}
}

func (s *s3TestSuite) TestDelete() {
	testcases := []struct {
		Case     string
		MockFunc func()
		ObjPaths []string
		Mime     e.ContentType
		ExpRes   bool
		ExpErr   error
	}{
		{
			Case: "Normal delete",
			MockFunc: func() {
				defaultS3api.On("DeleteObjects", mock.Anything, &awsS3.DeleteObjectsInput{
					Bucket: aws.String(defaultBucket),
					Delete: &types.Delete{
						Objects: []types.ObjectIdentifier{
							{Key: aws.String("data/foo.json")},
						},
						Quiet: aws.Bool(false),
					},
				}, mock.Anything).Return(&awsS3.DeleteObjectsOutput{
					Deleted: []types.DeletedObject{
						{Key: aws.String("data/foo.json")},
					},
				}, nil).Once()
			},
			ObjPaths: []string{"https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/data/foo.json"},
			Mime:     e.JSON,
			ExpRes:   true,
			ExpErr:   nil,
		},
		{
			Case:     "Region mismatch",
			MockFunc: func() {},
			ObjPaths: []string{"https://s3-ap-east-1.amazonaws.com/hsuanshao-bucket/data/foo.json"},
			Mime:     e.JSON,
			ExpRes:   false,
			ExpErr:   e.ErrWithoutPermissionToAccess,
		},
		{
			Case: "DeleteObjects failed",
			MockFunc: func() {
				defaultS3api.On("DeleteObjects", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("DeleteError")).Once()
			},
			ObjPaths: []string{"https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/data/foo.json"},
			Mime:     e.JSON,
			ExpRes:   false,
			ExpErr:   e.ErrDeleteObject,
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		res, err := s.CSP.Delete(mockCTX, c.Mime, c.ObjPaths)

		s.Equal(c.ExpRes, res, fmt.Sprintf("Case[%v]%v: check result", idx, c.Case))
		s.Equal(c.ExpErr, err, fmt.Sprintf("Case[%v]%v: check error", idx, c.Case))

		s.TearDownSuite()
	}
}

func (s *s3TestSuite) TestHealth() {
	testcases := []struct {
		Case     string
		MockFunc func()
		ExpRes   e.HealthStatus
		ExpErr   error
	}{
		{
			Case: "Healthy",
			MockFunc: func() {
				dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
					mockConn := new(MockConn)
					mockConn.On("Close").Return(nil)
					return mockConn, nil
				}
			},
			ExpRes: e.HealthStatus{Cloud: e.AWS},
			ExpErr: nil,
		},
		{
			Case: "Unhealthy",
			MockFunc: func() {
				dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
					return nil, errors.New("timeout")
				}
			},
			ExpRes: e.HealthStatus{Cloud: e.AWS, Latency: 30 * time.Second},
			ExpErr: e.ErrS3HealthTimeOut,
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		res, err := s.CSP.Health(mockCTX)

		s.Equal(c.ExpRes.Cloud, res.Cloud, fmt.Sprintf("Case[%v]%v: check cloud", idx, c.Case))
		if c.ExpErr != nil {
			s.Equal(c.ExpRes.Latency, res.Latency, fmt.Sprintf("Case[%v]%v: check latency", idx, c.Case))
		}
		s.Equal(c.ExpErr, err, fmt.Sprintf("Case[%v]%v: check error", idx, c.Case))

		s.TearDownSuite()
	}
}

func (s *s3TestSuite) TestNewS3() {
	roleArn := "arn:aws:iam::123456789012:role/test-role"
	conf := &e.Config{
		Region: defaultRegion,
		Bucket: defaultBucket,
		Option: &e.ConnectOption{
			RoleARN: &roleArn,
		},
	}
	s3srv, err := NewS3(mockCTX, conf)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), s3srv)
}

func (s *s3TestSuite) TestNewS3WithoutRoleARN() {
	conf := &e.Config{
		Region: defaultRegion,
		Bucket: defaultBucket,
	}
	s3srv, err := NewS3(mockCTX, conf)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), s3srv)
}

func (s *s3TestSuite) TestProcessObjURLV1() {
	impl := s.CSP.(*s3impl)
	region, bucket, key := impl.processObjURL("https://hsuanshao-bucket.s3.amazonaws.com/data/foo.json")
	s.Equal(defaultRegion, region)
	s.Equal("hsuanshao-bucket", bucket)
	s.Equal("data/foo.json", key)
}

func (s *s3TestSuite) TestProcessObjURLV2() {
	impl := s.CSP.(*s3impl)
	region, bucket, key := impl.processObjURL("https://s3-ap-northeast-1.amazonaws.com/hsuanshao-bucket/config/test.json")
	s.Equal("ap-northeast-1", region)
	s.Equal("hsuanshao-bucket", bucket)
	s.Equal("config/test.json", key)
}

func (s *s3TestSuite) TestProcessObjURLPlainKey() {
	impl := s.CSP.(*s3impl)
	region, bucket, key := impl.processObjURL("data/foo.json")
	s.Equal(defaultRegion, region)
	s.Equal(defaultBucket, bucket)
	s.Equal("data/foo.json", key)
}

func (s *s3TestSuite) TestGetObjURLWithoutDot() {
	impl := s.CSP.(*s3impl)
	url := impl.getObjURL("data/foo.json")
	s.Equal("https://hsuanshao-bucket.s3.amazonaws.com/data/foo.json", url)
}

func (s *s3TestSuite) TestGetObjURLWithDot() {
	impl := &s3impl{
		s3Srv:  defaultS3api,
		region: "ap-south-1",
		bucket: "hsuanshao.ag",
	}
	url := impl.getObjURL("/data/foo.json")
	s.Equal("https://s3-ap-south-1.amazonaws.com/hsuanshao.ag/data/foo.json", url)
}

func (s *s3TestSuite) TestGetObjURLWithLeadingSlash() {
	impl := s.CSP.(*s3impl)
	url := impl.getObjURL("/data/foo.json")
	s.Equal("https://hsuanshao-bucket.s3.amazonaws.com/data/foo.json", url)
}

func (s *s3TestSuite) TestReadObjectContentRegionMismatch() {
	_, _, err := s.CSP.ReadObjectContent(mockCTX, "https://s3-us-west-2.amazonaws.com/hsuanshao-bucket/data/foo.json")
	s.Equal(e.ErrWithoutPermissionToAccess, err)
}

func (s *s3TestSuite) TestReadObjectContentBucketMismatch() {
	_, _, err := s.CSP.ReadObjectContent(mockCTX, "https://wrong-bucket.s3.amazonaws.com/data/foo.json")
	s.Equal(e.ErrWithoutPermissionToAccess, err)
}

func (s *s3TestSuite) TestClose() {
	s.CSP.Close() // should not panic
}
