package cloud

import (
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/hsuanshao/golibs/ctx"

	"github.com/hsuanshao/golibs/buckets/cloud/provider"
	cpi "github.com/hsuanshao/golibs/buckets/cloud/provider/interface"
	e "github.com/hsuanshao/golibs/buckets/entity"
	bif "github.com/hsuanshao/golibs/buckets/interface"
)

// NewWriter is the constructor to launch bucket writer
func NewWriter(ctx ctx.CTX, conf *e.Config) (wr bif.BucketWriter, err error) {

	var blobService cpi.ObjectServiceProvider

	switch conf.CSP {
	case e.AWS:
		blobService, err = provider.NewS3(ctx, conf)
	case e.Minio:
		blobService, err = provider.NewMinio(ctx, conf)
	case e.Azure:
		blobService, err = provider.NewAzure(ctx, conf)
	case e.GCP:
		blobService, err = provider.NewGCS(ctx, conf)
	default:
		err = e.ErrInvalidateCSP
	}

	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "csp": conf.CSP}).Error("initial blob services by given config failed")
		return nil, err
	}

	return &writeImpl{
		cloud: blobService,
	}, nil
}

type writeImpl struct {
	cloud cpi.ObjectServiceProvider
}

// Upload to upload a new object,
//
//	possible error: if given objSavePath already been exists object,
//	                upload object mime not match
//	                other possible issue: duplicated copy to mulitple cloud failed, but it will only have a warning log, would not return error
func (wim *writeImpl) Upload(ctx ctx.CTX, mime e.ContentType, objSavePath string, payload io.Reader, size int64, objmetadata map[string]string) (objStorePath string, fiveMinsPresignedReadURL string, err error) {
	objURL, readPresignedURL, err := wim.cloud.Upload(ctx, mime, objSavePath, payload, size, objmetadata)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "content-type": mime, "objPath": objSavePath, "payload-size": size, "metadata": objmetadata}).Error("upload object failed")
		return "", "", e.ErrUploadObj
	}
	return objURL, readPresignedURL, nil
}

// Update to overwrite/update an exsits objects,
// general use case, update i18n translation file
func (wim *writeImpl) Update(ctx ctx.CTX, mime e.ContentType, objPath string, payload io.Reader, size int64, objmetadata map[string]string) (objURL string, err error) {
	objURL, err = wim.cloud.Override(ctx, mime, objPath, payload, size, objmetadata)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "content-type": mime, "objPath": objPath, "payload-size": size, "objmetadata": objmetadata}).Error("override original object from object storeage service failed")
		return "", e.ErrOverrideObj
	}

	return objURL, nil
}

// GeneratePutPresignedURL to gen a put (upload, if file exists, it will overwrite it) object permission URL, FE/Mob can applied this URL for user upload
// NOTE: suggestion, avoid use this function, because it will skip system workflow check, upload file directly
func (wim *writeImpl) GeneratePutPresignedURL(ctx ctx.CTX, uploadLimitDuration time.Duration, mime e.ContentType, newObjSavePath string, objectMetadata map[string]string) (putPresignedURL string, err error) {
	putPreURL, err := wim.cloud.PutPresignedURL(ctx, newObjSavePath, mime, uploadLimitDuration, objectMetadata)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "object metadata": objectMetadata, "objectPath": newObjSavePath}).Error("generate put object permission presigned url failed")
		return "", e.ErrGenPutPresignedURL
	}
	return putPreURL, nil
}

// DeleteObjects to delete object from bucket service
func (wim *writeImpl) DeleteObjects(ctx ctx.CTX, mime e.ContentType, objPathes []string) (res bool, err error) {
	res, err = wim.cloud.Delete(ctx, mime, objPathes)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "mime": mime, "objectPath": strings.Join(objPathes, ",")}).Error("delete object from blob storage failed")
		return false, err
	}

	return res, nil
}

// Close to Close bucketWriter agent
func (wim *writeImpl) Close() {
	wim.cloud.Close()
}
