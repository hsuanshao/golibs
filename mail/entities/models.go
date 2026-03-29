package entities

// Config holds the configuration for the AWS SES mail client.
type Config struct {
	// Region is the AWS region where SES is configured (e.g., "us-east-1").
	Region string `json:"region"`
	// Option holds optional advanced settings.
	Option *ConfigOption `json:"option,omitempty"`
}

// ConfigOption holds optional advanced configuration.
type ConfigOption struct {
	// RoleARN is the IAM Role ARN to assume before accessing SES.
	// If empty, the default credential chain is used.
	RoleARN         *string `json:"role_arn,omitempty"`
	AccessKeyID     *string `json:"access_key_id,omitempty"`
	SecretAccessKey *string `json:"secret_access_key,omitempty"`
}

// Mail is the structure for mail
type Mail struct {
	To      []string `json:"to"`
	Cc      []string `json:"cc,omitempty"`
	Bcc     []string `json:"bcc,omitempty"`
	From    string   `json:"from"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}
