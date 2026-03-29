package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gabriel-vasile/mimetype"
	"github.com/sirupsen/logrus"

	pifc "github.com/hsuanshao/golibs/buckets/cloud/provider/interface"
	e "github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

// S3ClientAPI defines the interface for S3 client operations
type S3ClientAPI interface {
	ListObjectsV2(ctx context.Context, params *awsS3.ListObjectsV2Input, optFns ...func(*awsS3.Options)) (*awsS3.ListObjectsV2Output, error)
	GetObject(ctx context.Context, params *awsS3.GetObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *awsS3.HeadObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *awsS3.PutObjectInput, optFns ...func(*awsS3.Options)) (*awsS3.PutObjectOutput, error)
	DeleteObjects(ctx context.Context, params *awsS3.DeleteObjectsInput, optFns ...func(*awsS3.Options)) (*awsS3.DeleteObjectsOutput, error)
	GetObjectAttributes(ctx context.Context, params *awsS3.GetObjectAttributesInput, optFns ...func(*awsS3.Options)) (*awsS3.GetObjectAttributesOutput, error)
}

// S3PresignClientAPI defines the interface for S3 presigning operations
type S3PresignClientAPI interface {
	PresignGetObject(ctx context.Context, params *awsS3.GetObjectInput, optFns ...func(*awsS3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	PresignPutObject(ctx context.Context, params *awsS3.PutObjectInput, optFns ...func(*awsS3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// NewS3 ...
func NewS3(ctx ctx.CTX, conf *e.Config) (s3srv pifc.ObjectServiceProvider, err error) {
	region, bucket := conf.Region, conf.Bucket

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "region": region}).Error("failed to load aws default config")
		return nil, err
	}

	if conf.Option != nil {
		if conf.Option.AccessKey != nil && conf.Option.SecretAccessKey != nil &&
			strings.TrimSpace(*conf.Option.AccessKey) != "" && strings.TrimSpace(*conf.Option.SecretAccessKey) != "" {
			cfg.Credentials = credentials.NewStaticCredentialsProvider(
				*conf.Option.AccessKey,
				*conf.Option.SecretAccessKey,
				"",
			)
		} else if conf.Option.RoleARN != nil && strings.TrimSpace(*conf.Option.RoleARN) != "" {
			stsClient := sts.NewFromConfig(cfg)
			creds := stscreds.NewAssumeRoleProvider(stsClient, *conf.Option.RoleARN)
			cfg.Credentials = aws.NewCredentialsCache(creds)
		}
	}

	client := awsS3.NewFromConfig(cfg)
	presignClient := awsS3.NewPresignClient(client)

	return &s3impl{
		s3Srv:         client,
		presignClient: presignClient,
		bucket:        bucket,
		region:        region,
		isMinio:       false,
	}, nil
}

type s3impl struct {
	s3Srv         S3ClientAPI
	presignClient S3PresignClientAPI
	bucket        string
	region        string
	isMinio       bool
}

var (
	// awsS3ObjUrl is object url pattern to aws s3
	// rul is https://{bucket name}.s3.amazonaws.com/{object key}
	awsS3ObjUrl = "https://%s.s3.amazonaws.com/%s"

	// awsS3RegionObjUrl is the pattern while bucket has ".",
	// pattern is https://s3-{region}.amazonaws.com/{bucket name}/{object key}
	awsS3RegionObjUrl = "https://s3-%s.amazonaws.com/%s/%s"
)

// GetObjectList to fetch object list
func (s3Im *s3impl) GetObjectList(ctx ctx.CTX, prefix, delim string) (objURLs []string, err error) {
	// NOTE: please well manage object storage apporch, because pull
	// object list, return up to 1000.
	// to pull object list also cost some money, therefore, to put
	// object without planning is danger, and costly
	listInput := &awsS3.ListObjectsV2Input{
		Bucket: aws.String(s3Im.bucket),
	}

	if prefix != "" && strings.TrimSpace(prefix) != "" {
		listInput.Prefix = aws.String(prefix)
	}

	if delim != "" && strings.TrimSpace(delim) != "" {
		listInput.Delimiter = aws.String(delim)
	}

	output, err := s3Im.s3Srv.ListObjectsV2(ctx, listInput)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "bucket": s3Im.bucket, "prefix": prefix, "delim": delim}).Error("list objects from s3 failed")
		return nil, e.ErrFetchObjList
	}

	if len(output.Contents) == 0 {
		ctx.WithFields(logrus.Fields{"prefix": prefix, "delim": delim, "listInput": listInput}).Warn("list object with zero object output content")
	}

	for idx, obj := range output.Contents {
		if obj.Key == nil {
			ctx.WithFields(logrus.Fields{"idx": idx, "obj": obj}).Warn("one of object key is nil")
			continue
		}

		url := s3Im.getObjURL(*obj.Key)
		objURLs = append(objURLs, url)
	}

	return objURLs, nil
}

func (s3Im *s3impl) getObjURL(objKey string) (url string) {
	prefixChk := strings.HasPrefix(objKey, "/")
	if prefixChk {
		objKey = objKey[1:]
	}
	hasDot := strings.ContainsAny(s3Im.bucket, ".")
	switch hasDot {
	case true:
		url = fmt.Sprintf(awsS3RegionObjUrl, s3Im.region, s3Im.bucket, objKey)
	case false:
		url = fmt.Sprintf(awsS3ObjUrl, s3Im.bucket, objKey)
	}
	return url
}

func (s3Im *s3impl) processObjURL(url string) (region, bucket, fileKey string) {
	region = s3Im.region
	bucket = s3Im.bucket

	prefixOk := strings.HasPrefix(url, "https://")
	if prefixOk {
		url = url[8:]
	}
	var urlV1ok, urlV2ok, urlOK bool
	urlOK = false
	urlV1ok = strings.Contains(url, ".s3.amazonaws.com/")
	if !urlV1ok {
		urlV2ok = strings.Contains(url, "amazonaws.com/")
	}
	var splitArr []string
	if urlV2ok || urlV1ok {
		urlOK = true
		if urlV1ok {
			splitArr = strings.Split(url, ".s3.amazonaws.com/")
			bucket = splitArr[0]
			splitArr = append([]string{""}, strings.Split(splitArr[1], "/")...)
		}

		if urlV2ok {
			splitArr = strings.Split(url, ".amazonaws.com/")
			region = splitArr[0][3:]
			splitArr = strings.Split(splitArr[1], "/")
			bucket = splitArr[0]
		}

		l := len(splitArr[1:]) - 1
		for n, s := range splitArr[1:] {
			fileKey += s
			if n < l {
				fileKey += "/"
			}
		}
	}

	if !prefixOk && !urlOK {
		fileKey = url
	}

	return region, bucket, fileKey
}

// ReadObjectContent to read object content
func (s3Im *s3impl) ReadObjectContent(ctx ctx.CTX, objectPath string) (objReader io.ReadCloser, metadata map[string]string, err error) {
	objRegion, objBucket, objKey := s3Im.processObjURL(objectPath)
	if objRegion != s3Im.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return nil, nil, e.ErrWithoutPermissionToAccess
	}

	if objBucket != s3Im.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return nil, nil, e.ErrWithoutPermissionToAccess
	}

	response, err := s3Im.s3Srv.GetObject(ctx, &awsS3.GetObjectInput{Bucket: aws.String(s3Im.bucket), Key: aws.String(objKey)})
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "region": s3Im.region, "bucket": s3Im.bucket, "key": objKey, "response": response}).Error("get object from s3 failed")
		return nil, nil, e.ErrGetObjFromS3
	}
	// Do not close Body here, pass it to caller

	objMeta := map[string]string{}
	for key, val := range response.Metadata {
		objMeta[key] = val
	}
	contenType := ""
	if response.ContentType != nil {
		contenType = *response.ContentType
	}
	objSizeInByte := int64(0)
	if response.ContentLength != nil {
		objSizeInByte = *response.ContentLength
	}
	ctx.WithFields(logrus.Fields{"content type": contenType, "metadata": objMeta, "objBytes": objSizeInByte}).Info("check object context")

	// Return Body directly for streaming
	return response.Body, objMeta, nil
}

// IsObjectExists to check object existence by given url
func (s3Im *s3impl) IsObjectExists(ctx ctx.CTX, objURL string) (existed bool, err error) {
	objRegion, objBucket, objKey := s3Im.processObjURL(objURL)
	if objRegion != s3Im.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return false, e.ErrWithoutPermissionToAccess
	}

	if objBucket != s3Im.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return false, e.ErrWithoutPermissionToAccess
	}

	_, err = s3Im.s3Srv.HeadObject(ctx, &awsS3.HeadObjectInput{Bucket: aws.String(s3Im.bucket), Key: aws.String(objKey)})
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "bucket": s3Im.bucket, "objKey": objKey, "objURL": objURL}).Info("head object get error return")
		// NOTE: here doesn't return error is due to if given objURL not exists, get error to this function is a kind of positive result, kept log for tracing, learning
		return false, nil
	}

	return true, nil
}

// GenReadPresignedURL to generate a private object with ttl permission
func (s3Im *s3impl) GenReadPresignedURL(ctx ctx.CTX, objURL string, duration time.Duration) (readPresignedURL string, err error) {
	objRegion, objBucket, objKey := s3Im.processObjURL(objURL)
	if objRegion != s3Im.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return "", e.ErrWithoutPermissionToAccess
	}

	if objBucket != s3Im.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return "", e.ErrWithoutPermissionToAccess
	}

	request, err := s3Im.presignClient.PresignGetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(s3Im.bucket),
		Key:    aws.String(objKey),
	}, func(o *awsS3.PresignOptions) {
		o.Expires = duration
	})

	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "bucket": s3Im.bucket, "objectKey": objKey, "objPath": objURL}).Error("try genereate read presigned url failed")
		return "", e.ErrGenReadPresignedURL
	}

	return request.URL, nil
}

// PutPresignedURL to generate an object upload with a ttl permision
func (s3Im *s3impl) PutPresignedURL(ctx ctx.CTX, objURL string, mime e.ContentType, duration time.Duration, metaData map[string]string) (presignedURL string, err error) {
	objRegion, objBucket, objKey := s3Im.processObjURL(objURL)
	if objRegion != s3Im.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return "", e.ErrWithoutPermissionToAccess
	}

	if objBucket != s3Im.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return "", e.ErrWithoutPermissionToAccess
	}

	objExists, _ := s3Im.IsObjectExists(ctx, objURL)
	if objExists {
		ctx.WithFields(logrus.Fields{"objURL": objURL, "metadata": metaData, "content-type": mime}).Warn("try to generate by given obj url, however the url already exists an objects")
		return "", e.ErrObjectPathHasItem
	}

	request, err := s3Im.presignClient.PresignPutObject(ctx, &awsS3.PutObjectInput{
		Bucket:      aws.String(s3Im.bucket),
		Key:         aws.String(objKey),
		ContentType: aws.String(mime.String()),
	}, func(o *awsS3.PresignOptions) {
		o.Expires = duration
	})

	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "objBucket": objBucket, "objKey": objKey}).Error("generate put presign url failed")
		return "", e.ErrGenPutPresignedURL
	}
	return request.URL, nil
}

// Upload to upload object
func (s3Im *s3impl) Upload(ctx ctx.CTX, ct e.ContentType, objpath string, payload io.Reader, size int64, objmetadata map[string]string) (URL string, readPresignedURL string, err error) {
	objRegion, objBucket, objKey := s3Im.processObjURL(objpath)
	if objRegion != s3Im.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return "", "", e.ErrWithoutPermissionToAccess
	}

	if objBucket != s3Im.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return "", "", e.ErrWithoutPermissionToAccess
	}

	exists, _ := s3Im.IsObjectExists(ctx, objpath)

	if exists {
		ctx.WithFields(logrus.Fields{"objRegion": objRegion, "objBucket": objBucket, "objKey": objKey, "given objPath": objpath}).Warn("given obj path already has object")
		return "", "", e.ErrObjectPathHasItem
	}

	return s3Im.uploadCore(ctx, ct, objpath, objKey, payload, size, objmetadata)
}

// Override to override exists obj, this function will compare original content and new content
func (s3Im *s3impl) Override(ctx ctx.CTX, ct e.ContentType, objPath string, payload io.Reader, size int64, objmetadata map[string]string) (objURL string, err error) {
	objRegion, objBucket, objKey := s3Im.processObjURL(objPath)
	if objRegion != s3Im.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return "", e.ErrWithoutPermissionToAccess
	}

	if objBucket != s3Im.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return "", e.ErrWithoutPermissionToAccess
	}

	// fetch original object information
	response, err := s3Im.s3Srv.HeadObject(ctx, &awsS3.HeadObjectInput{Bucket: aws.String(s3Im.bucket), Key: aws.String(objKey)})
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "region": s3Im.region, "bucket": s3Im.bucket, "key": objKey}).Error("get object from s3 failed")
		return "", e.ErrFetchOriginObjFromS3ByGivenObjPath
	}

	if response.LastModified == nil {
		ctx.WithFields(logrus.Fields{"err": err, "region": s3Im.region, "bucket": s3Im.bucket, "key": objKey, "last modified time": response.LastModified}).Warn("get object from s3 but without obj modified time")
		return "", e.ErrFetchOriginObjFromS3ByGivenObjPath
	}

	originContentType := ""
	if response.ContentType != nil {
		originContentType = *response.ContentType
	}

	var latestModifiedTime time.Time
	if response.LastModified != nil {
		latestModifiedTime = *response.LastModified
		latestModifiedTime.UTC()
	}
	originTimeStr := latestModifiedTime.Format(time.RFC3339)
	ctx.WithFields(logrus.Fields{"origin-content-type": originContentType, "latest-modified-utc-time": originTimeStr}).Info("[record] origin object related information")

	// uploadCore handles the upload without checking existence first
	url, _, err := s3Im.uploadCore(ctx, ct, objPath, objKey, payload, size, objmetadata)
	return url, err
}

func (s3Im *s3impl) uploadCore(ctx ctx.CTX, ct e.ContentType, objPath, objKey string, payload io.Reader, size int64, objmetadata map[string]string) (URL string, readPresignedURL string, err error) {
	// Peeking first 512 bytes for MIME detection without fully consuming the reader
	head := make([]byte, 512)
	n, err := payload.Read(head)
	if err != nil && err != io.EOF {
		ctx.WithField("err", err).Error("failed to read payload head for mime detection")
		return "", "", e.ErrUploadObj
	}
	// Reconstruct the reader
	payloadWithHead := io.MultiReader(bytes.NewReader(head[:n]), payload)

	objContentType := mimetype.Detect(head[:n])
	// Use explicit content type from argument if it's application/octet-stream (default/unknown)
	// or if we trust the input more. But the logic here compares them.
	// Let's keep existing logic: valid if detected matches expected.
	// NOTE: mimetype.Detect might return specific types while passed ct is generic.
	// For now, strict check as per original code.
	if !e.IsMatchContentType(objContentType.String(), ct) {
		ctx.WithFields(logrus.Fields{"expect content type": ct, "detect content type": objContentType}).Error("upload new object content type is not match input parameter")
		return "", "", e.ErrUploadNotMatchContentType
	}

	// Use generic io.Reader Body
	// Note: AWS SDK uses Seekable behavior if possible to calculate hash, but works with non-seekable.
	// However, without size, it might buffer. Pass ContentLength if we have it (size > 0).
	putInput := &awsS3.PutObjectInput{
		Bucket:      aws.String(s3Im.bucket),
		Key:         aws.String(objKey),
		Body:        payloadWithHead,
		ContentType: aws.String(ct.String()),
		Metadata:    objmetadata,
	}
	// AWS SDK v2 requires ContentLength for non-seekable streams to avoid buffering everything in memory to calculate it?
	// Actually v2 is smarter, but providing ContentLength is best practice if known.
	if size > 0 {
		putInput.ContentLength = aws.Int64(size)
	}

	putOutput, err := s3Im.s3Srv.PutObject(ctx, putInput)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "bucket": s3Im.bucket, "objectPath": objPath, "obj": objKey}).Error("upload object to s3 bucket failed")
		return "", "", e.ErrUploadObjToS3
	}

	verID := ""
	// NOTE: VersionId not nil only while the bucket been settle has versioning control
	if putOutput.VersionId != nil {
		verID = *putOutput.VersionId
	}
	ctx.WithField("objVerID", verID).Info("check object version id")

	objURL := s3Im.getObjURL(objKey)
	readPresignedURL, err = s3Im.GenReadPresignedURL(ctx, objURL, 5*time.Minute)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "objURL": objURL}).Warn("try to genenerate read presigned url failed from just upload object url")
		// NOTE: return nil err here, is due to some cloud object storeage service might has latency issue to generate presigned url (read)
		return objURL, "", nil
	}

	return objURL, readPresignedURL, nil
}

// Delete for delete object item from blob object storage
func (s3Im *s3impl) Delete(ctx ctx.CTX, contentType e.ContentType, objPathes []string) (result bool, err error) {
	var objects []types.ObjectIdentifier
	region := s3Im.region
	bucket := s3Im.bucket
	for _, objPath := range objPathes {
		objRegion, objBucket, objKey := s3Im.processObjURL(objPath)
		if objRegion != region {
			ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
			return false, e.ErrWithoutPermissionToAccess
		}

		if objBucket != bucket {
			ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
			return false, e.ErrWithoutPermissionToAccess
		}
		objects = append(objects, types.ObjectIdentifier{
			Key: aws.String(objKey),
		})
	}

	resOpt, err := s3Im.s3Srv.DeleteObjects(ctx, &awsS3.DeleteObjectsInput{
		Bucket: aws.String(s3Im.bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false),
		},
	})
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err}).Error("delete object from s3 failed")
		return false, e.ErrDeleteObject
	}

	if len(resOpt.Deleted) != len(objects) {
		ctx.WithFields(logrus.Fields{"response output delete object length": len(resOpt.Deleted), "input objects": len(objects)}).Warn("delete object count doesn't matching")
	}

	return true, nil
}

// dialTimeout is a variable to allow mocking net.DialTimeout in tests
var dialTimeout = net.DialTimeout

// Health to tell every platform service latency
func (s3Im *s3impl) Health(ctx ctx.CTX) (status e.HealthStatus, err error) {
	now := time.Now()
	endpoint := "s3." + s3Im.region + ".amazonaws.com:443"
	timeout := time.Duration(30 * time.Second)
	conn, err := dialTimeout("tcp", endpoint, timeout)
	if err != nil {
		return e.HealthStatus{Cloud: e.AWS, Latency: timeout}, e.ErrS3HealthTimeOut
	}
	if conn != nil {
		conn.Close()
	}

	spendTime := time.Since(now)
	return e.HealthStatus{
		Cloud:   e.AWS,
		Latency: spendTime,
	}, nil
}

// Close to close client
func (s3Im *s3impl) Close() {
	// NOTE: s3Im.s3Srv is a aws sesssion, not a s3 client connection
	// it has no needed to close it.
}
