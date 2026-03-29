package entity

import (
	"strings"

	"github.com/hsuanshao/golibs/ctx"
)

// CloudServiceProvider defines blob bucket service provider
type CloudServiceProvider string

const (
	// AWS is the aws s3
	AWS CloudServiceProvider = "aws"
	// GCP: is the GCS
	GCP CloudServiceProvider = "gcp"
	// Azure is the Azure storage
	Azure CloudServiceProvider = "azure"
	// Minio is a customized blob storage
	Minio CloudServiceProvider = "minio"
)

var (
	mapCSP = map[string]CloudServiceProvider{
		"aws":   AWS,
		"gcp":   GCP,
		"azure": Azure,
		"minio": Minio,
	}
)

// IsValidateCloudServiceProvider ...
func IsValidateCloudServiceProvider(ctx ctx.CTX, cspName string) bool {
	if _, ok := mapCSP[strings.ToLower(cspName)]; !ok {
		ctx.WithField("csp name", cspName).Warn("given csp name unable map object storage service provider")
		return false
	}
	return true
}

// Config describe a bucket service configuration parameters
type Config struct {
	CSP      CloudServiceProvider `json:"cloud"`
	Region   string               `json:"region"`
	Bucket   string               `json:"bucket"`
	Priority int                  `json:"priority"`
	Option   *ConnectOption       `json:"option,omitempty"`
}

// ConnectOption ...
type ConnectOption struct {
	ConnectCredential []byte  `json:"connect_credential,omitempty"`
	RoleARN           *string `json:"role_arn,omitempty"`
	AccessKey         *string `json:"access_key,omitempty"`
	SecretAccessKey   *string `json:"secret_access_key,omitempty"`
	Endpoint          *string `json:"endpoint,omitempty"`
}
