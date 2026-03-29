package mqtt

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	awsIot "github.com/aws/aws-sdk-go-v2/service/iotdataplane"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"

	"github.com/hsuanshao/golibs/ctx"
	m "github.com/hsuanshao/golibs/mqtt/entities"
)

// ─── AWS IoT Data Plane abstraction (for mocking in tests) ────────────────────

// IoTDataPlaneAPI defines the subset of the iotdataplane.Client used by this
// package.  Wrapping the concrete AWS client in an interface allows tests to
// inject a mock without making real network calls.
type IoTDataPlaneAPI interface {
	Publish(ctx context.Context, params *awsIot.PublishInput, optFns ...func(*awsIot.Options)) (*awsIot.PublishOutput, error)
}

// ─── Paho MQTT client abstraction (for mocking in tests) ─────────────────────

// PahoClientAPI defines the subset of pahomqtt.Client used by this package.
type PahoClientAPI interface {
	Connect() pahomqtt.Token
	Disconnect(quiesce uint)
	Publish(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token
	Subscribe(topic string, qos byte, callback pahomqtt.MessageHandler) pahomqtt.Token
	IsConnected() bool
}

// ─── Constructor ──────────────────────────────────────────────────────────────

// New creates an AWS IoT Core MQTT client.
//
// Publishing is done via the AWS IoT Data Plane HTTP API (iotdataplane.Publish),
// which allows the service to authenticate using IAM/SigV4 without requiring a
// persistent TCP/TLS connection.
//
// Subscribing is done over MQTT-over-WebSocket-Secure (MQTT/WSS) using the
// Paho client library.  The WSS URL is signed with AWS Signature Version 4
// using the same credentials resolved from the AWS SDK configuration.
//
// Call Connect before using Publish or Subscribe.
func New(c ctx.CTX, conf *m.Config) (m.MQTTClient, m.MQTTError) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	awsCfg, err := config.LoadDefaultConfig(c, config.WithRegion(conf.Region))
	if err != nil || conf.Option == nil || conf.Option.RoleARN == nil || strings.TrimSpace(*conf.Option.RoleARN) == "" {
		c.WithFields(logrus.Fields{"err": err, "region": conf.Region}).
			Error("mqtt: failed to load aws default config")
		return nil, m.ErrInvalidConfig
	}

	// Optional: assume an IAM role before accessing IoT services.
	if conf.Option != nil && conf.Option.RoleARN != nil &&
		strings.TrimSpace(*conf.Option.RoleARN) != "" {
		stsClient := sts.NewFromConfig(awsCfg)
		creds := stscreds.NewAssumeRoleProvider(stsClient, *conf.Option.RoleARN)
		awsCfg.Credentials = aws.NewCredentialsCache(creds)
	}

	// IoT Data Plane client: used exclusively for Publish over HTTP/SigV4.
	iotEndpoint := fmt.Sprintf("https://%s", conf.Endpoint)
	iotClient := awsIot.NewFromConfig(awsCfg, func(o *awsIot.Options) {
		o.BaseEndpoint = aws.String(iotEndpoint)
	})

	return &impl{
		conf:      conf,
		awsCfg:    awsCfg,
		iotClient: iotClient,
	}, nil
}

// ─── impl ─────────────────────────────────────────────────────────────────────

type impl struct {
	conf      *m.Config
	awsCfg    aws.Config
	iotClient IoTDataPlaneAPI

	mu         sync.RWMutex
	pahoClient PahoClientAPI

	// pahoClientFactory allows tests to inject a mock Paho factory.
	pahoClientFactory func(opts *pahomqtt.ClientOptions) PahoClientAPI
}

// pahoFactory is the default factory that creates a real Paho MQTT client.
func pahoFactory(opts *pahomqtt.ClientOptions) PahoClientAPI {
	return pahomqtt.NewClient(opts)
}

// ─── Connect ──────────────────────────────────────────────────────────────────

// Connect establishes the underlying MQTT/WSS connection to AWS IoT Core.
// It must be called before Publish or Subscribe.
func (c *impl) Connect(ctx ctx.CTX) m.MQTTError {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build a SigV4-signed WSS URL for the Paho MQTT client.
	signedURL, err := c.buildSignedWSSURL(ctx)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err}).Error("mqtt: failed to build signed wss url")
		return m.ErrConnectFailed
	}

	opts := pahomqtt.NewClientOptions().
		AddBroker(signedURL).
		SetClientID(c.conf.ClientID).
		SetCleanSession(true).
		SetKeepAlive(time.Duration(c.conf.KeepAlive) * time.Second).
		SetConnectTimeout(time.Duration(c.conf.ConnectTimeout) * time.Second).
		SetAutoReconnect(false) // Re-signing is required on reconnect; callers should reconnect explicitly.

	factory := c.pahoClientFactory
	if factory == nil {
		factory = pahoFactory
	}

	pahoClient := factory(opts)
	token := pahoClient.Connect()
	if !token.WaitTimeout(time.Duration(c.conf.ConnectTimeout) * time.Second) {
		ctx.WithFields(logrus.Fields{"client_id": c.conf.ClientID}).
			Error("mqtt: connect token timed out")
		return m.ErrConnectFailed
	}
	if token.Error() != nil {
		ctx.WithFields(logrus.Fields{"err": token.Error(), "client_id": c.conf.ClientID}).
			Error("mqtt: paho connect failed")
		return m.ErrConnectFailed
	}

	c.pahoClient = pahoClient
	ctx.WithFields(logrus.Fields{"client_id": c.conf.ClientID, "endpoint": c.conf.Endpoint}).
		Info("mqtt: connected to aws iot core")
	return nil
}

// ─── Disconnect ───────────────────────────────────────────────────────────────

// Disconnect closes the MQTT/WSS connection gracefully.
func (c *impl) Disconnect(ctx ctx.CTX) m.MQTTError {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pahoClient == nil || !c.pahoClient.IsConnected() {
		ctx.WithFields(logrus.Fields{"client_id": c.conf.ClientID}).
			Warn("mqtt: disconnect called but client is not connected")
		return nil
	}

	c.pahoClient.Disconnect(250)
	c.pahoClient = nil
	ctx.WithFields(logrus.Fields{"client_id": c.conf.ClientID}).Info("mqtt: disconnected")
	return nil
}

// ─── Publish ──────────────────────────────────────────────────────────────────

// Publish sends a message to an IoT Core topic using the AWS Data Plane HTTP
// API.  This does NOT require the paho MQTT connection to be active.
func (c *impl) Publish(ctx ctx.CTX, topic string, payload []byte) m.MQTTError {
	if strings.TrimSpace(topic) == "" {
		ctx.WithFields(logrus.Fields{"topic": topic}).Error("mqtt: publish topic is empty")
		return m.ErrPublishFailed
	}

	input := &awsIot.PublishInput{
		Topic:   aws.String(topic),
		Payload: payload,
		Qos:     int32(c.conf.QoS),
		Retain:  c.conf.Retained,
	}

	_, err := c.iotClient.Publish(ctx.Context, input)
	if err != nil {
		ctx.WithFields(logrus.Fields{
			"err":   err,
			"topic": topic,
		}).Error("mqtt: iotdataplane publish failed")
		return m.ErrPublishFailed
	}

	ctx.WithFields(logrus.Fields{
		"topic":        topic,
		"payload_size": len(payload),
	}).Info("mqtt: message published via iotdataplane")
	return nil
}

// ─── Subscribe ────────────────────────────────────────────────────────────────

// Subscribe registers a message handler for the given MQTT topic pattern.
// Connect must have been called successfully before Subscribe.
func (c *impl) Subscribe(ctx ctx.CTX, topic string, handler func(ctx.CTX, string, []byte) m.MQTTError) m.MQTTError {
	c.mu.RLock()
	pahoClient := c.pahoClient
	c.mu.RUnlock()

	if pahoClient == nil || !pahoClient.IsConnected() {
		ctx.WithFields(logrus.Fields{"topic": topic}).
			Error("mqtt: subscribe called but client is not connected")
		return m.ErrNotConnected
	}

	if strings.TrimSpace(topic) == "" {
		ctx.WithFields(logrus.Fields{"topic": topic}).Error("mqtt: subscribe topic is empty")
		return m.ErrSubscribeFailed
	}

	msgHandler := func(_ pahomqtt.Client, msg pahomqtt.Message) {
		if err := handler(ctx, msg.Topic(), msg.Payload()); err != nil {
			ctx.WithFields(logrus.Fields{
				"err":   err,
				"topic": msg.Topic(),
			}).Error("mqtt: message handler returned error")
		}
	}

	token := pahoClient.Subscribe(topic, byte(c.conf.QoS), msgHandler)
	if !token.WaitTimeout(time.Duration(c.conf.ConnectTimeout) * time.Second) {
		ctx.WithFields(logrus.Fields{"topic": topic}).Error("mqtt: subscribe token timed out")
		return m.ErrSubscribeFailed
	}
	if token.Error() != nil {
		ctx.WithFields(logrus.Fields{"err": token.Error(), "topic": topic}).
			Error("mqtt: paho subscribe failed")
		return m.ErrSubscribeFailed
	}

	ctx.WithFields(logrus.Fields{"topic": topic}).Info("mqtt: subscribed to topic")
	return nil
}

// ─── SigV4 WSS URL builder ────────────────────────────────────────────────────

// buildSignedWSSURL constructs an AWS Signature Version 4 pre-signed WebSocket
// URL that the Paho client uses to authenticate to AWS IoT Core.
//
// Reference: https://docs.aws.amazon.com/iot/latest/developerguide/mqtt-ws.html
func (c *impl) buildSignedWSSURL(ctx context.Context) (string, error) {
	creds, err := c.awsCfg.Credentials.Retrieve(ctx)
	if err != nil {
		return "", fmt.Errorf("retrieve credentials: %w", err)
	}

	now := time.Now().UTC()
	datestamp := now.Format("20060102")       // YYYYMMDD
	amzdate := now.Format("20060102T150405Z") // ISO‑8601 basic format

	region := c.conf.Region
	service := "iotdevicegateway"
	algorithm := "AWS4-HMAC-SHA256"
	host := c.conf.Endpoint
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, region, service)

	// ── 1. Canonical query string ─────────────────────────────────────────────
	queryParams := url.Values{}
	queryParams.Set("X-Amz-Algorithm", algorithm)
	queryParams.Set("X-Amz-Credential", creds.AccessKeyID+"/"+credentialScope)
	queryParams.Set("X-Amz-Date", amzdate)
	queryParams.Set("X-Amz-SignedHeaders", "host")

	if creds.SessionToken != "" {
		queryParams.Set("X-Amz-Security-Token", creds.SessionToken)
	}

	canonicalQueryString := queryParams.Encode()

	// ── 2. Canonical request ──────────────────────────────────────────────────
	canonicalURI := "/mqtt"
	canonicalHeaders := fmt.Sprintf("host:%s\n", host)
	signedHeaders := "host"
	payloadHash := sha256hex([]byte(""))

	canonicalRequest := strings.Join([]string{
		"GET",
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// ── 3. String to sign ─────────────────────────────────────────────────────
	stringToSign := strings.Join([]string{
		algorithm,
		amzdate,
		credentialScope,
		sha256hex([]byte(canonicalRequest)),
	}, "\n")

	// ── 4. Signing key ────────────────────────────────────────────────────────
	signingKey := hmacSHA256(
		hmacSHA256(
			hmacSHA256(
				hmacSHA256(
					[]byte("AWS4"+creds.SecretAccessKey),
					datestamp,
				),
				region,
			),
			service,
		),
		"aws4_request",
	)

	// ── 5. Signature ──────────────────────────────────────────────────────────
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	queryParams.Set("X-Amz-Signature", signature)

	// Re-encode after adding signature (url.Values.Encode() sorts keys).
	signedURL := fmt.Sprintf("wss://%s%s?%s", host, canonicalURI, queryParams.Encode())
	return signedURL, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func validateConfig(conf *m.Config) m.MQTTError {
	if conf == nil {
		return m.ErrInvalidConfig
	}
	if strings.TrimSpace(conf.Endpoint) == "" {
		return m.ErrInvalidConfig
	}
	if strings.TrimSpace(conf.Region) == "" {
		return m.ErrInvalidConfig
	}
	if strings.TrimSpace(conf.ClientID) == "" {
		return m.ErrInvalidConfig
	}
	if conf.QoS < 0 || conf.QoS > 1 {
		// AWS IoT Core supports QoS 0 and 1 only.
		return m.ErrInvalidConfig
	}
	if conf.KeepAlive <= 0 {
		return m.ErrInvalidConfig
	}
	if conf.ConnectTimeout <= 0 {
		return m.ErrInvalidConfig
	}
	if conf.Option != nil {
		if conf.Option.RoleARN == nil {
			return m.ErrInvalidConfig
		}
		if strings.TrimSpace(*conf.Option.RoleARN) == "" {
			return m.ErrInvalidConfig
		}
	}
	return nil
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
