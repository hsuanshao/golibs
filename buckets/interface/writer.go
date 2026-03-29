package ifc

import (
	"io"
	"time"

	"github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

// BucketWriter privde methods to upload, overwrite, delete object on bucket
type BucketWriter interface {
	// Upload to upload a new object,
	//  possible error: if given objSavePath already been exists object,
	//                  upload object mime not match
	//                  other possible issue: duplicated copy to mulitple cloud failed, but it will only have a warning log, would not return error
	Upload(ctx ctx.CTX, mime entity.ContentType, objSavePath string, payload io.Reader, size int64, objmetadata map[string]string) (objStorePath string, fiveMinsPresignedReadURL string, err error)
	// Update to overwrite/update an exsits objects,
	// general use case, update i18n translation file, if you try update an obj with a non exists objpath it will return error
	Update(ctx ctx.CTX, mime entity.ContentType, objPath string, payload io.Reader, size int64, objmetadata map[string]string) (objURL string, err error)
	// GeneratePutPresignedURL to gen a put (upload, if file exists, it will overwrite it) object permission URL, FE/Mob can applied this URL for user upload
	// NOTE: suggestion, avoid use this function, because it will skip system workflow check, upload file directly
	GeneratePutPresignedURL(ctx ctx.CTX, uploadLimitDuration time.Duration, mime entity.ContentType, newObjSavePath string, objectMetadata map[string]string) (putPresignedURL string, err error)
	// DeleteObjects response to delete object from blob storage
	DeleteObjects(ctx ctx.CTX, mime entity.ContentType, objPathes []string) (res bool, err error)
	// Close to Close bucketWriter agent
	Close()
}
