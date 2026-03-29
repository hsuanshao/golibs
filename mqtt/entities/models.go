package entities

// Config holds the configuration for the AWS IoT Core MQTT client.
type Config struct {
	// Endpoint is the AWS IoT Core data plane endpoint.
	// Obtain it by running: aws iot describe-endpoint --endpoint-type iot:Data-ATS
	// Example: "abcdefg1234567.iot.us-east-1.amazonaws.com"
	Endpoint string `json:"endpoint"`

	// Region is the AWS region of your IoT Core deployment (e.g., "us-east-1").
	Region string `json:"region"`

	// ClientID is the MQTT client identifier. Must be unique per concurrent connection.
	ClientID string `json:"client_id"`

	// QoS is the Quality of Service level for publish and subscribe operations.
	// Allowed values: 0 (at most once), 1 (at least once).
	// AWS IoT Core does not support QoS 2.
	QoS int `json:"qos"`

	// Retained controls whether the broker retains the last published message
	// for each topic. AWS IoT Core supports retained messages.
	Retained bool `json:"retained"`

	// KeepAlive is the number of seconds between keep-alive pings sent over the
	// MQTT WebSocket connection. Must be > 0. Recommended: 30.
	KeepAlive int `json:"keep_alive"`

	// ConnectTimeout is the maximum number of seconds to wait for the MQTT
	// WebSocket handshake to complete.
	ConnectTimeout int `json:"connect_timeout"`

	// Option holds optional advanced settings such as IAM Role assumption.
	Option *ConfigOption `json:"option,omitempty"`
}

// ConfigOption holds optional advanced configuration.
type ConfigOption struct {
	// RoleARN is the IAM Role ARN to assume before calling AWS IoT APIs.
	// If nil or empty, the default credential chain is used.
	RoleARN *string `json:"role_arn,omitempty"`
}
