package entity

import "errors"

var (
	// ErrNotImpl to tell calll this function not ready to apply
	ErrNotImpl = errors.New("this function not finish implmentation yet")

	// ErrNilConf means bucket config parameter is nil
	ErrNilConf = errors.New("config parameter is nil")

	// ErrS3BucketisEmptyStr means bucket in config is empty string
	ErrS3BucketisEmptyStr = errors.New("bucket name is empty string")

	// ErrS3RegionisEmptyStr means region in config is empty string
	ErrS3RegionisEmptyStr = errors.New("region name is empty string")

	// ErrUnknownOSP means user given Cloud Object storage service provider is not in our listing, which means our code not support yet
	ErrUnknownOSP = errors.New("unknow or not support object storage service provider")

	// ErrFetchObjList means that based on configuration and given condition, we get an error return from cloud SDK, usually is due to given dondition get empty object unit listing
	ErrFetchObjList = errors.New("fetch object list failed")

	// ErrWithoutPermissionToAccess means input Object URL/Object Key Path belong to other region or bucket, that we have no permission to access it
	ErrWithoutPermissionToAccess = errors.New("without permission to access object")

	// ErrGetObjFromS3 ...
	ErrGetObjFromS3 = errors.New("get object from s3 but get error response")

	// ErrReadObjContent ...
	ErrReadObjContent = errors.New("ioutil read content get unexpected error")

	// ErrS3HealthTimeOut ...
	ErrS3HealthTimeOut = errors.New("s3 region endpoint health check seems timeout")

	// ErrObjectPathHasItem ...
	ErrObjectPathHasItem = errors.New("target put object path already been used by another object")

	// ErrGenPutPresignedURL ...
	ErrGenPutPresignedURL = errors.New("generate put presigned url failed")

	// ErrUploadNotMatchContentType ...
	ErrUploadNotMatchContentType = errors.New("upload failed is not match settle mime content type")

	// ErrUploadObjToS3 ...
	ErrUploadObjToS3 = errors.New("upload object to s3 bucket failed")

	// ErrGenReadPresignedURL ...
	ErrGenReadPresignedURL = errors.New("generated read presigned url failed")

	// ErrOverrideObject ...
	ErrOverrideObject = errors.New("override object failed")

	// ErrFetchOriginObjFromS3ByGivenObjPath ...
	ErrFetchOriginObjFromS3ByGivenObjPath = errors.New("unable fetch any object from given object path which is going to override")

	// ErrFetchOriginObject is a cross-provider alias for ErrFetchOriginObjFromS3ByGivenObjPath.
	// Use this in non-S3 providers (GCP, Azure) for clarity.
	ErrFetchOriginObject = ErrFetchOriginObjFromS3ByGivenObjPath

	// ErrUploadObj ...
	ErrUploadObj = errors.New("object upload to object storage service failed")

	// ErrOverrideObj ...
	ErrOverrideObj = errors.New("override original object by new object failed")

	// ErrInitS3Client ...
	ErrInitS3Client = errors.New("initialize s3 client failed")

	// ErrInitBucketReader ...
	ErrInitBucketReader = errors.New("unable init cloud bucket reader by given configuration")

	// ErrInitBucketWriter ...
	ErrInitBucketWriter = errors.New("unable init cloud bucket writer by given configuration")

	// ErrBucketKeyEncryptDisabled ...
	ErrBucketKeyEncryptDisabled = errors.New("the uploaded object uses an S3 Bucket Key for server-side encryption with Amazon Web Services KMS was disabled")

	// ErrEnablMinioHostIsRequired ...
	ErrEnablMinioHostIsRequired = errors.New("config host is required while object storagte service is minio")

	// ErrEnableMinioAccessKeyIsRequired ...
	ErrEnableMinioAccessKeyIsRequired = errors.New("to enable minio, config access ky is required")

	// ErrEnableMinioSecretAccessKeyIsRequired ...
	ErrEnableMinioSecretAccessKeyIsRequired = errors.New("to enable minio, config secret access key is required")

	// ErrInitMinioClient ....
	ErrInitMinioClient = errors.New("initial minio client failed")

	// ErrPreCheckBucket ...
	ErrPreCheckBucket = errors.New("pre check bucket exists status, but get error return")

	// ErrBucketNotExistsInRegion ...
	ErrBucketNotExistsInRegion = errors.New("bucket not exists in the region")

	// ErrDeleteObject ...
	ErrDeleteObject = errors.New("delete object from blob service failed")

	// ErrNilMinioEndpointURL ...
	ErrNilMinioEndpointURL = errors.New("lack of minio blob endpoint from config document")

	// ErrInvalidateCSP ...
	ErrInvalidateCSP = errors.New("invalidated csp enum value")
)
