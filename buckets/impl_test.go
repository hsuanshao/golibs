package buckets

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hsuanshao/golibs/buckets/entity"
	"github.com/hsuanshao/golibs/ctx"
)

func TestInitNilConfigReturnsNil(t *testing.T) {
	c := ctx.Background()
	svc := Init(c, nil)
	assert.Nil(t, svc, "Init with nil config should return nil")
}

func TestInitValidConfigReturnsService(t *testing.T) {
	c := ctx.Background()
	conf := &entity.Config{
		CSP:    entity.AWS,
		Region: "ap-northeast-1",
		Bucket: "test-bucket",
	}
	svc := Init(c, conf)
	assert.NotNil(t, svc, "Init with valid config should return a non-nil Service")
}

func TestGetBucketReaderAzureNotImpl(t *testing.T) {
	c := ctx.Background()
	conf := &entity.Config{
		CSP:    entity.Azure,
		Region: "eastus",
		Bucket: "test-container",
	}
	svc := Init(c, conf)
	assert.NotNil(t, svc)

	reader, err := svc.GetBucketReader(c)
	assert.Nil(t, reader)
	assert.Equal(t, entity.ErrInitBucketReader, err)
}

func TestGetBucketWriterAzureNotImpl(t *testing.T) {
	c := ctx.Background()
	conf := &entity.Config{
		CSP:    entity.Azure,
		Region: "eastus",
		Bucket: "test-container",
	}
	svc := Init(c, conf)
	assert.NotNil(t, svc)

	writer, err := svc.GetBucketWriter(c)
	assert.Nil(t, writer)
	assert.Equal(t, entity.ErrInitBucketWriter, err)
}

func TestGetBucketReaderInvalidCSP(t *testing.T) {
	c := ctx.Background()
	conf := &entity.Config{
		CSP:    entity.CloudServiceProvider("oracle"),
		Region: "us-east-1",
		Bucket: "test-bucket",
	}
	svc := Init(c, conf)
	assert.NotNil(t, svc)

	reader, err := svc.GetBucketReader(c)
	assert.Nil(t, reader)
	assert.Equal(t, entity.ErrInitBucketReader, err)
}

func TestGetBucketWriterInvalidCSP(t *testing.T) {
	c := ctx.Background()
	conf := &entity.Config{
		CSP:    entity.CloudServiceProvider("oracle"),
		Region: "us-east-1",
		Bucket: "test-bucket",
	}
	svc := Init(c, conf)
	assert.NotNil(t, svc)

	writer, err := svc.GetBucketWriter(c)
	assert.Nil(t, writer)
	assert.Equal(t, entity.ErrInitBucketWriter, err)
}
