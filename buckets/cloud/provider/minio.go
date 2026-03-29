package provider

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go"
	"github.com/sirupsen/logrus"

	pifc "github.com/hsuanshao/golibs/buckets/cloud/provider/interface"
	e "github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

// MinioClient defines the interface for minio operations to facilitate testing
type MinioClient interface {
	PutObject(bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (n int64, err error)
	GetObject(bucketName, objectName string, opts minio.GetObjectOptions) (io.ReadCloser, error)
	StatObject(bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	RemoveObjects(bucketName string, objectsCh <-chan string) <-chan minio.RemoveObjectError
	PresignedGetObject(bucketName, objectName string, expires time.Duration, reqParams url.Values) (u *url.URL, err error)
	PresignedPutObject(bucketName, objectName string, expires time.Duration) (u *url.URL, err error)
	ListObjects(bucketName, objectPrefix string, recursive bool, doneCh <-chan struct{}) <-chan minio.ObjectInfo
	ListBuckets() ([]minio.BucketInfo, error)
}

// Wrapper for real minio client to satisfy MinioClient interface
type minioClientWrapper struct {
	client *minio.Client
}

func (w *minioClientWrapper) PutObject(bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	return w.client.PutObject(bucketName, objectName, reader, objectSize, opts)
}

func (w *minioClientWrapper) GetObject(bucketName, objectName string, opts minio.GetObjectOptions) (io.ReadCloser, error) {
	// minio.Client.GetObject returns (*minio.Object, error)
	// *minio.Object implements io.ReadCloser
	return w.client.GetObject(bucketName, objectName, opts)
}

func (w *minioClientWrapper) StatObject(bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	return w.client.StatObject(bucketName, objectName, opts)
}

func (w *minioClientWrapper) RemoveObjects(bucketName string, objectsCh <-chan string) <-chan minio.RemoveObjectError {
	return w.client.RemoveObjects(bucketName, objectsCh)
}

func (w *minioClientWrapper) PresignedGetObject(bucketName, objectName string, expires time.Duration, reqParams url.Values) (u *url.URL, err error) {
	return w.client.PresignedGetObject(bucketName, objectName, expires, reqParams)
}

func (w *minioClientWrapper) PresignedPutObject(bucketName, objectName string, expires time.Duration) (u *url.URL, err error) {
	return w.client.PresignedPutObject(bucketName, objectName, expires)
}

func (w *minioClientWrapper) ListObjects(bucketName, objectPrefix string, recursive bool, doneCh <-chan struct{}) <-chan minio.ObjectInfo {
	return w.client.ListObjects(bucketName, objectPrefix, recursive, doneCh)
}

func (w *minioClientWrapper) ListBuckets() ([]minio.BucketInfo, error) {
	return w.client.ListBuckets()
}

// NewMinio ...
func NewMinio(ctx ctx.CTX, conf *e.Config) (bktSrc pifc.ObjectServiceProvider, err error) {
	region, bucket := conf.Region, conf.Bucket

	// force rewrite is due to minio default region is 'us-east-1'
	// LINK: https://github.com/minio/minio/discussions/15063
	if region != "us-east-1" {
		region = "us-east-1"
	}

	endpoint := ""
	if conf.Option == nil || conf.Option.Endpoint == nil || strings.TrimSpace(*conf.Option.Endpoint) == "" {
		ctx.Error("initial minio blob api but lack of endpoint")
		return nil, e.ErrNilMinioEndpointURL
	}
	endpoint = strings.TrimSpace(*conf.Option.Endpoint)

	// minio-go v6 initialization
	useSSL := true
	if strings.HasPrefix(endpoint, "http://") {
		useSSL = false
		endpoint = strings.TrimPrefix(endpoint, "http://")
	} else if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
	}

	minioClient, err := minio.New(endpoint, *conf.Option.AccessKey, *conf.Option.SecretAccessKey, useSSL)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "endpoint": endpoint}).Error("initialize minio client failed")
		return nil, err
	}

	return &minioImpl{
		client:   &minioClientWrapper{client: minioClient},
		bucket:   bucket,
		region:   region,
		endpoint: endpoint,
	}, nil
}

type minioImpl struct {
	client   MinioClient
	bucket   string
	region   string
	endpoint string
}

var (
	// minioObjURL is object url pattern to minio blob storage
	// rule is https://{endpoint}/{bucket name}/{object key}
	minioObjURL = "https://%s/%s/%s"
)

// GetObjectList to fetch object list
func (minioIm *minioImpl) GetObjectList(ctx ctx.CTX, prefix, delim string) (objURLs []string, err error) {
	// Create a done channel to control 'ListObjects' go routine.
	doneCh := make(chan struct{})
	defer close(doneCh)

	recursive := true
	if delim != "" {
		recursive = false
	}

	isRecursive := recursive
	objectCh := minioIm.client.ListObjects(minioIm.bucket, prefix, isRecursive, doneCh)

	for obj := range objectCh {
		if obj.Err != nil {
			ctx.WithFields(logrus.Fields{"err": obj.Err, "bucket": minioIm.bucket}).Error("list objects error")
			return nil, e.ErrFetchObjList
		}
		if obj.Key == "" {
			continue
		}
		url := minioIm.getObjURL(obj.Key)
		objURLs = append(objURLs, url)
	}

	return objURLs, nil
}

func (minioIm *minioImpl) getObjURL(objKey string) (url string) {
	prefixChk := strings.HasPrefix(objKey, "/")
	if prefixChk {
		objKey = objKey[1:]
	}
	url = fmt.Sprintf(minioObjURL, minioIm.endpoint, minioIm.bucket, objKey)
	return url
}

func (minioIm *minioImpl) processObjURL(url string) (region, bucket, fileKey string) {
	region = minioIm.region
	bucket = minioIm.bucket

	prefixOk := strings.HasPrefix(url, "https://")
	if prefixOk {
		url = url[8:]
	}
	var urlV1ok, urlOK bool
	urlOK = false
	urlV1ok = strings.Contains(url, minioIm.endpoint)

	var splitArr []string
	if urlV1ok {
		urlOK = true
		if urlV1ok {
			splitArr = strings.Split(url, minioIm.endpoint)
			pathParts := strings.Split(strings.TrimPrefix(splitArr[1], "/"), "/")
			if len(pathParts) > 0 {
				bucket = pathParts[0]
				splitArr = pathParts
			}
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
func (minioIm *minioImpl) ReadObjectContent(ctx ctx.CTX, objectPath string) (objReader io.ReadCloser, metadata map[string]string, err error) {
	objRegion, objBucket, objKey := minioIm.processObjURL(objectPath)
	if objRegion != minioIm.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return nil, nil, e.ErrWithoutPermissionToAccess
	}

	if objBucket != minioIm.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return nil, nil, e.ErrWithoutPermissionToAccess
	}

	// Fetch metadata via StatObject first
	objInfo, err := minioIm.client.StatObject(minioIm.bucket, objKey, minio.StatObjectOptions{})
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "key": objKey}).Error("stat object failed")
		return nil, nil, e.ErrGetObjFromS3
	}

	object, err := minioIm.client.GetObject(minioIm.bucket, objKey, minio.GetObjectOptions{})
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "region": minioIm.region, "bucket": minioIm.bucket, "key": objKey}).Error("get object from minio failed")
		return nil, nil, e.ErrGetObjFromS3
	}

	objMeta := map[string]string{}

	for k, v := range objInfo.Metadata {
		if len(v) > 0 {
			objMeta[k] = v[0]
		}
	}

	contenType := objInfo.ContentType
	if contenType != "" {
		objMeta["Content-Type"] = contenType
	}

	ctx.WithFields(logrus.Fields{"content type": contenType, "metadata": objMeta, "objBytes": objInfo.Size}).Info("check object context")

	return object, objMeta, nil
}

// IsObjectExists to check object existence by given url
func (minioIm *minioImpl) IsObjectExists(ctx ctx.CTX, objURL string) (existed bool, err error) {
	objRegion, objBucket, objKey := minioIm.processObjURL(objURL)
	if objRegion != minioIm.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return false, e.ErrWithoutPermissionToAccess
	}

	if objBucket != minioIm.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return false, e.ErrWithoutPermissionToAccess
	}

	_, err = minioIm.client.StatObject(minioIm.bucket, objKey, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		ctx.WithFields(logrus.Fields{"err": err, "bucket": minioIm.bucket, "objKey": objKey, "objURL": objURL}).Info("stat object error")
		return false, nil
	}

	return true, nil
}

// GenReadPresginedURL to generate a private object with ttl permission
func (minioIm *minioImpl) GenReadPresignedURL(ctx ctx.CTX, objURL string, duration time.Duration) (readPresignedURL string, err error) {
	objRegion, objBucket, objKey := minioIm.processObjURL(objURL)
	if objRegion != minioIm.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return "", e.ErrWithoutPermissionToAccess
	}

	if objBucket != minioIm.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return "", e.ErrWithoutPermissionToAccess
	}

	u, err := minioIm.client.PresignedGetObject(minioIm.bucket, objKey, duration, nil)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "bucket": minioIm.bucket, "objectKey": objKey, "objPath": objURL}).Error("try genereate read presigned url failed")
		return "", e.ErrGenReadPresignedURL
	}

	return u.String(), nil
}

// PutPresignedURL to generate an object upload with a ttl permision
func (minioIm *minioImpl) PutPresignedURL(ctx ctx.CTX, objURL string, mime e.ContentType, duration time.Duration, metaData map[string]string) (presignedURL string, err error) {
	objRegion, objBucket, objKey := minioIm.processObjURL(objURL)
	if objRegion != minioIm.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return "", e.ErrWithoutPermissionToAccess
	}

	if objBucket != minioIm.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return "", e.ErrWithoutPermissionToAccess
	}

	objExists, _ := minioIm.IsObjectExists(ctx, objURL)
	if objExists {
		ctx.WithFields(logrus.Fields{"objURL": objURL, "metadata": metaData, "content-type": mime}).Warn("try to generate by given obj url, however the url already exists an objects")
		return "", e.ErrObjectPathHasItem
	}

	u, err := minioIm.client.PresignedPutObject(minioIm.bucket, objKey, duration)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "objBucket": objBucket, "objKey": objKey}).Error("generate put presign url failed")
		return "", e.ErrGenPutPresignedURL
	}
	return u.String(), nil
}

// Upload to upload object
func (minioIm *minioImpl) Upload(ctx ctx.CTX, ct e.ContentType, objpath string, payload io.Reader, size int64, objmetadata map[string]string) (URL string, readPresignedURL string, err error) {
	objRegion, objBucket, objKey := minioIm.processObjURL(objpath)
	if objRegion != minioIm.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return "", "", e.ErrWithoutPermissionToAccess
	}

	if objBucket != minioIm.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return "", "", e.ErrWithoutPermissionToAccess
	}

	exists, _ := minioIm.IsObjectExists(ctx, objpath)
	if exists {
		ctx.WithFields(logrus.Fields{"objRegion": objRegion, "objBucket": objBucket, "objKey": objKey, "given objPath": objpath}).Warn("given obj path already has object")
		return "", "", e.ErrObjectPathHasItem
	}

	return minioIm.uploadCore(ctx, ct, objpath, objKey, payload, size, objmetadata)
}

// Override to override exists obj, this function will compare original content and new content
func (minioIm *minioImpl) Override(ctx ctx.CTX, ct e.ContentType, objPath string, payload io.Reader, size int64, objmetadata map[string]string) (objURL string, err error) {
	objRegion, objBucket, objKey := minioIm.processObjURL(objPath)
	if objRegion != minioIm.region {
		ctx.WithFields(logrus.Fields{"object region": objRegion}).Warn("given object path region is not in permission region")
		return "", e.ErrWithoutPermissionToAccess
	}

	if objBucket != minioIm.bucket {
		ctx.WithFields(logrus.Fields{"object bucket": objBucket}).Warn("given object path bucket is not in expected bucket")
		return "", e.ErrWithoutPermissionToAccess
	}

	// fetch original object information
	objInfo, err := minioIm.client.StatObject(minioIm.bucket, objKey, minio.StatObjectOptions{})
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "region": minioIm.region, "bucket": minioIm.bucket, "key": objKey}).Error("get object from minio failed")
		return "", e.ErrFetchOriginObjFromS3ByGivenObjPath
	}

	originContentType := objInfo.ContentType

	originTimeStr := objInfo.LastModified.Format(time.RFC3339)
	ctx.WithFields(logrus.Fields{"origin-content-type": originContentType, "latest-modified-utc-time": originTimeStr}).Info("[record] origin object related information")

	optObjURL, _, err := minioIm.uploadCore(ctx, ct, objPath, objKey, payload, size, objmetadata)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "content-type": ct, "objPath": objPath}).Error("upload new object content failed")
		return "", e.ErrOverrideObject
	}
	return optObjURL, nil
}

func (minioIm *minioImpl) uploadCore(ctx ctx.CTX, ct e.ContentType, objPath, objKey string, payload io.Reader, size int64, objmetadata map[string]string) (URL string, readPresignedURL string, err error) {
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
	if !e.IsMatchContentType(objContentType.String(), ct) {
		ctx.WithFields(logrus.Fields{"expect content type": ct, "detect content type": objContentType}).Error("upload new object content type is not match input parameter")
		return "", "", e.ErrUploadNotMatchContentType
	}

	// Prepare PutObjectOptions
	opts := minio.PutObjectOptions{
		ContentType:  ct.String(),
		UserMetadata: objmetadata,
	}

	_, err = minioIm.client.PutObject(minioIm.bucket, objKey, payloadWithHead, size, opts)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "bucket": minioIm.bucket, "objectPath": objPath, "obj": objKey}).Error("upload object to minio failed")
		return "", "", e.ErrUploadObjToS3
	}

	objURL := minioIm.getObjURL(objKey)
	readPresignedURL, err = minioIm.GenReadPresignedURL(ctx, objURL, 5*time.Minute)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "objURL": objURL}).Warn("try to genenerate read presigned url failed from just upload object url")
		return objURL, "", nil
	}

	return objURL, readPresignedURL, nil
}

// Delete for delete object item from blob object storage
func (minioIm *minioImpl) Delete(ctx ctx.CTX, contentType e.ContentType, objPathes []string) (result bool, err error) {
	// RemoveObjects takes a channel of object names
	objectsCh := make(chan string)
	go func() {
		defer close(objectsCh)
		for _, objPath := range objPathes {
			_, _, objKey := minioIm.processObjURL(objPath)
			objectsCh <- objKey
		}
	}()

	errorCh := minioIm.client.RemoveObjects(minioIm.bucket, objectsCh)

	hasErr := false
	for e := range errorCh {
		if e.Err != nil {
			ctx.WithField("err", e.Err).Error("failed to remove object")
			hasErr = true
		}
	}

	if hasErr {
		return false, e.ErrDeleteObject
	}

	return true, nil
}

// Health to tell every platform service latency
func (minioIm *minioImpl) Health(ctx ctx.CTX) (status e.HealthStatus, err error) {
	now := time.Now()
	// Use ListBuckets as a health check
	_, err = minioIm.client.ListBuckets()
	if err != nil {
		ctx.WithField("err", err).Error("minio health check failed")
		return e.HealthStatus{Cloud: e.Minio, Latency: 30 * time.Second}, e.ErrS3HealthTimeOut
	}

	spendTime := time.Since(now)
	return e.HealthStatus{
		Cloud:   e.Minio,
		Latency: spendTime,
	}, nil
}

// Close to close client
func (minioIm *minioImpl) Close() {
	// minio-go client doesn't need explicit close (re-uses http transport)
}
