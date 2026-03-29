package ifc

import "github.com/hsuanshao/golibs/ctx"

type Service interface {
	// GetBucketReader to get BukcetReader methods based on configuration setup
	GetBucketReader(ctx ctx.CTX) (reader BucketReader, err error)
	// GetBucketWriter to get BucketWriter methods based on configuration setup
	GetBucketWriter(ctx ctx.CTX) (writer BucketWriter, err error)
}
