package mqtt

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsCredentials "github.com/aws/aws-sdk-go-v2/credentials"
	awsIot "github.com/aws/aws-sdk-go-v2/service/iotdataplane"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/hsuanshao/golibs/ctx"
	m "github.com/hsuanshao/golibs/mqtt/entities"
)

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	testEndpoint     = "abc123.iot.us-east-1.amazonaws.com"
	testRegion       = "us-east-1"
	testClientID     = "test-client-001"
	testTopic        = "devices/sensor/data"
	testRoleARN      = "arn:aws:iam::123456789012:role/IoTRole"
	errMsgUnexpected = "Case[%d] %s: unexpected error"
	errMsgClientNil  = "Case[%d] %s: client nil-ness mismatch"
)

// defaultConf returns a fully-valid config suitable for use in tests.
func defaultConf() *m.Config {
	return &m.Config{
		Endpoint:       testEndpoint,
		Region:         testRegion,
		ClientID:       testClientID,
		QoS:            1,
		Retained:       false,
		KeepAlive:      30,
		ConnectTimeout: 5,
	}
}

// ─── Mock: IoTDataPlaneAPI ────────────────────────────────────────────────────

type MockIoTClient struct{ mock.Mock }

func (mc *MockIoTClient) Publish(c context.Context, params *awsIot.PublishInput, optFns ...func(*awsIot.Options)) (*awsIot.PublishOutput, error) {
	args := mc.Called(c, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*awsIot.PublishOutput), args.Error(1)
}

// ─── Mock: PahoClientAPI ─────────────────────────────────────────────────────

type MockPahoClient struct{ mock.Mock }

func (mc *MockPahoClient) Connect() pahomqtt.Token {
	return mc.Called().Get(0).(pahomqtt.Token)
}
func (mc *MockPahoClient) Disconnect(quiesce uint) { mc.Called(quiesce) }
func (mc *MockPahoClient) Publish(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token {
	return mc.Called(topic, qos, retained, payload).Get(0).(pahomqtt.Token)
}
func (mc *MockPahoClient) Subscribe(topic string, qos byte, callback pahomqtt.MessageHandler) pahomqtt.Token {
	return mc.Called(topic, qos, callback).Get(0).(pahomqtt.Token)
}
func (mc *MockPahoClient) IsConnected() bool { return mc.Called().Bool(0) }

// ─── Mock: pahomqtt.Token ─────────────────────────────────────────────────────

type mockToken struct {
	err     error
	timeout bool
}

func (t *mockToken) Wait() bool                       { return true }
func (t *mockToken) WaitTimeout(_ time.Duration) bool { return !t.timeout }
func (t *mockToken) Done() <-chan struct{}            { ch := make(chan struct{}); close(ch); return ch }
func (t *mockToken) Error() error                     { return t.err }

func okToken() pahomqtt.Token         { return &mockToken{} }
func errToken(e error) pahomqtt.Token { return &mockToken{err: e} }
func timeoutToken() pahomqtt.Token    { return &mockToken{timeout: true} }

// ─── Mock: pahomqtt.Message ──────────────────────────────────────────────────

type mockMessage struct {
	topic   string
	payload []byte
}

func (mm *mockMessage) Duplicate() bool   { return false }
func (mm *mockMessage) Qos() byte         { return 0 }
func (mm *mockMessage) Retained() bool    { return false }
func (mm *mockMessage) Topic() string     { return mm.topic }
func (mm *mockMessage) MessageID() uint16 { return 0 }
func (mm *mockMessage) Payload() []byte   { return mm.payload }
func (mm *mockMessage) Ack()              {}

// ─── Suite ────────────────────────────────────────────────────────────────────

var mockCTX = ctx.Background()

type mqttTestSuite struct {
	suite.Suite
	iotClient  *MockIoTClient
	pahoClient *MockPahoClient
}

func TestMQTTSuite(t *testing.T) {
	ctx.SetDebugLevel()
	suite.Run(t, new(mqttTestSuite))
}

func (s *mqttTestSuite) SetupTest() {
	mockCTX = ctx.Background()
	s.iotClient = new(MockIoTClient)
	s.pahoClient = new(MockPahoClient)
}

// buildImpl constructs an *impl with mocked IoT and Paho clients injected.
// The pahoClientFactory is wired to return s.pahoClient on every invocation so
// tests can assert on it without touching real network code.
func (s *mqttTestSuite) buildImpl() *impl {
	paho := s.pahoClient
	return &impl{
		conf: defaultConf(),
		awsCfg: aws.Config{
			Credentials: awsCredentials.NewStaticCredentialsProvider("AKID", "SECRET", ""),
		},
		iotClient: s.iotClient,
		pahoClientFactory: func(_ *pahomqtt.ClientOptions) PahoClientAPI {
			return paho
		},
	}
}

// ─── Test: validateConfig ─────────────────────────────────────────────────────

func (s *mqttTestSuite) TestValidateConfig() {
	roleARN := testRoleARN
	emptyStr := ""
	blankStr := "   "

	testcases := []struct {
		Case   string
		Conf   *m.Config
		ExpErr error
	}{
		// ── nil / missing required fields ──────────────────────────────────────
		{
			Case:   "nil config",
			Conf:   nil,
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "empty endpoint",
			Conf:   &m.Config{Endpoint: "", Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "blank endpoint",
			Conf:   &m.Config{Endpoint: "   ", Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "empty region",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: "", ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "blank region",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: "   ", ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "empty clientID",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: "", QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "blank clientID",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: "  ", QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: m.ErrInvalidConfig,
		},
		// ── QoS bounds ─────────────────────────────────────────────────────────
		{
			Case:   "QoS -1 is invalid",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: -1, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "QoS 2 is invalid (AWS IoT Core does not support it)",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: 2, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: m.ErrInvalidConfig,
		},
		// ── keepAlive / connectTimeout ─────────────────────────────────────────
		{
			Case:   "KeepAlive 0 is invalid",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 0, ConnectTimeout: 5},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "ConnectTimeout 0 is invalid",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 0},
			ExpErr: m.ErrInvalidConfig,
		},
		// ── Option / RoleARN ───────────────────────────────────────────────────
		{
			Case:   "Option present with nil RoleARN",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5, Option: &m.ConfigOption{RoleARN: nil}},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "Option present with empty RoleARN",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5, Option: &m.ConfigOption{RoleARN: &emptyStr}},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "Option present with blank RoleARN",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5, Option: &m.ConfigOption{RoleARN: &blankStr}},
			ExpErr: m.ErrInvalidConfig,
		},
		// ── valid cases ────────────────────────────────────────────────────────
		{
			Case:   "valid: QoS 0, no Option",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: nil,
		},
		{
			Case:   "valid: QoS 1, no Option",
			Conf:   &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: 1, KeepAlive: 30, ConnectTimeout: 5},
			ExpErr: nil,
		},
		{
			Case: "valid: with RoleARN",
			Conf: &m.Config{
				Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID,
				QoS: 1, KeepAlive: 30, ConnectTimeout: 5,
				Option: &m.ConfigOption{RoleARN: &roleARN},
			},
			ExpErr: nil,
		},
	}

	for idx, c := range testcases {
		err := validateConfig(c.Conf)
		s.Equal(c.ExpErr, err, fmt.Sprintf(errMsgUnexpected, idx, c.Case))
	}
}

// ─── Test: New ────────────────────────────────────────────────────────────────
//
// config.LoadDefaultConfig, iotdataplane.NewFromConfig, sts.NewFromConfig, and
// stscreds.NewAssumeRoleProvider all construct in-memory structs only — no
// network calls are made.  AWS credential resolution is lazy, so these tests
// are safe to run without real AWS credentials.
func (s *mqttTestSuite) TestNew() {
	roleARN := testRoleARN

	testcases := []struct {
		Case         string
		Conf         *m.Config
		ExpClientNil bool
		ExpErr       error
	}{
		{
			Case:         "nil config → ErrInvalidConfig",
			Conf:         nil,
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			Case:         "empty endpoint → ErrInvalidConfig",
			Conf:         &m.Config{Endpoint: "", Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			Case:         "empty region → ErrInvalidConfig",
			Conf:         &m.Config{Endpoint: testEndpoint, Region: "", ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			Case:         "empty clientID → ErrInvalidConfig",
			Conf:         &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: "", QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			Case:         "valid config, no Option → success",
			Conf:         &m.Config{Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID, QoS: 0, KeepAlive: 30, ConnectTimeout: 5},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			Case: "valid config with RoleARN → assume-role wired, success",
			Conf: &m.Config{
				Endpoint: testEndpoint, Region: testRegion, ClientID: testClientID,
				QoS: 1, KeepAlive: 30, ConnectTimeout: 5,
				Option: &m.ConfigOption{RoleARN: &roleARN},
			},
			ExpClientNil: false,
			ExpErr:       nil,
		},
	}

	for idx, c := range testcases {
		client, err := New(mockCTX, c.Conf)
		s.Equal(c.ExpErr, err, fmt.Sprintf(errMsgUnexpected, idx, c.Case))
		s.Equal(c.ExpClientNil, client == nil, fmt.Sprintf(errMsgClientNil, idx, c.Case))
	}
}

// ─── Test: Connect ────────────────────────────────────────────────────────────

func (s *mqttTestSuite) TestConnect() {
	testcases := []struct {
		Case     string
		MockFunc func()
		ExpErr   error
	}{
		{
			Case: "connect succeeds",
			MockFunc: func() {
				s.pahoClient.On("Connect").Return(okToken()).Once()
			},
			ExpErr: nil,
		},
		{
			Case: "paho returns error token → ErrConnectFailed",
			MockFunc: func() {
				s.pahoClient.On("Connect").Return(errToken(errors.New("broker refused"))).Once()
			},
			ExpErr: m.ErrConnectFailed,
		},
		{
			Case: "paho connect times out → ErrConnectFailed",
			MockFunc: func() {
				s.pahoClient.On("Connect").Return(timeoutToken()).Once()
			},
			ExpErr: m.ErrConnectFailed,
		},
	}

	for idx, c := range testcases {
		s.SetupTest()
		i := s.buildImpl()
		c.MockFunc()

		err := i.Connect(mockCTX)
		s.Equal(c.ExpErr, err, fmt.Sprintf(errMsgUnexpected, idx, c.Case))
		s.pahoClient.AssertExpectations(s.T())
	}
}

// ─── Test: Disconnect ─────────────────────────────────────────────────────────

func (s *mqttTestSuite) TestDisconnect() {
	testcases := []struct {
		Case     string
		MockFunc func()
		PreWired bool // whether to inject a pahoClient into the impl before the call
		ExpErr   error
	}{
		{
			Case: "disconnects when paho reports connected",
			MockFunc: func() {
				s.pahoClient.On("IsConnected").Return(true).Once()
				s.pahoClient.On("Disconnect", uint(250)).Return().Once()
			},
			PreWired: true,
			ExpErr:   nil,
		},
		{
			Case: "no-op when paho reports not connected",
			MockFunc: func() {
				s.pahoClient.On("IsConnected").Return(false).Once()
			},
			PreWired: true,
			ExpErr:   nil,
		},
		{
			Case:     "no-op when pahoClient is nil (never connected)",
			MockFunc: func() {},
			PreWired: false,
			ExpErr:   nil,
		},
	}

	for idx, c := range testcases {
		s.SetupTest()
		i := s.buildImpl()
		c.MockFunc()

		if c.PreWired {
			i.pahoClient = s.pahoClient
		}

		err := i.Disconnect(mockCTX)
		s.Equal(c.ExpErr, err, fmt.Sprintf(errMsgUnexpected, idx, c.Case))
		s.pahoClient.AssertExpectations(s.T())
	}
}

// ─── Test: Publish ────────────────────────────────────────────────────────────

func (s *mqttTestSuite) TestPublish() {
	payload := []byte(`{"temp":42}`)

	testcases := []struct {
		Case     string
		Topic    string
		Payload  []byte
		MockFunc func()
		ExpErr   error
	}{
		{
			Case:    "publish succeeds",
			Topic:   testTopic,
			Payload: payload,
			MockFunc: func() {
				s.iotClient.On("Publish",
					mock.Anything,
					mock.MatchedBy(func(input *awsIot.PublishInput) bool {
						return *input.Topic == testTopic
					}),
					mock.Anything,
				).Return(&awsIot.PublishOutput{}, nil).Once()
			},
			ExpErr: nil,
		},
		{
			Case:    "iotdataplane returns error → ErrPublishFailed",
			Topic:   testTopic,
			Payload: payload,
			MockFunc: func() {
				s.iotClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("throttled")).Once()
			},
			ExpErr: m.ErrPublishFailed,
		},
		{
			Case:     "empty topic → ErrPublishFailed (no AWS call)",
			Topic:    "",
			Payload:  payload,
			MockFunc: func() {},
			ExpErr:   m.ErrPublishFailed,
		},
		{
			Case:     "blank topic → ErrPublishFailed (no AWS call)",
			Topic:    "   ",
			Payload:  payload,
			MockFunc: func() {},
			ExpErr:   m.ErrPublishFailed,
		},
		{
			Case:    "nil payload is allowed (empty message body)",
			Topic:   testTopic,
			Payload: nil,
			MockFunc: func() {
				s.iotClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).
					Return(&awsIot.PublishOutput{}, nil).Once()
			},
			ExpErr: nil,
		},
	}

	for idx, c := range testcases {
		s.SetupTest()
		i := s.buildImpl()
		c.MockFunc()

		err := i.Publish(mockCTX, c.Topic, c.Payload)
		s.Equal(c.ExpErr, err, fmt.Sprintf(errMsgUnexpected, idx, c.Case))
		s.iotClient.AssertExpectations(s.T())
	}
}

// ─── Test: Subscribe ──────────────────────────────────────────────────────────

func (s *mqttTestSuite) TestSubscribe() {
	noopHandler := func(_ ctx.CTX, _ string, _ []byte) m.MQTTError { return nil }

	testcases := []struct {
		Case      string
		Topic     string
		Connected bool // whether to pre-wire pahoClient into the impl
		MockFunc  func()
		ExpErr    error
	}{
		{
			Case:      "subscribe succeeds",
			Topic:     testTopic,
			Connected: true,
			MockFunc: func() {
				s.pahoClient.On("IsConnected").Return(true)
				s.pahoClient.On("Subscribe", testTopic, byte(1), mock.Anything).
					Return(okToken()).Once()
			},
			ExpErr: nil,
		},
		{
			Case:      "paho returns error token → ErrSubscribeFailed",
			Topic:     testTopic,
			Connected: true,
			MockFunc: func() {
				s.pahoClient.On("IsConnected").Return(true)
				s.pahoClient.On("Subscribe", testTopic, byte(1), mock.Anything).
					Return(errToken(errors.New("permission denied"))).Once()
			},
			ExpErr: m.ErrSubscribeFailed,
		},
		{
			Case:      "paho subscribe times out → ErrSubscribeFailed",
			Topic:     testTopic,
			Connected: true,
			MockFunc: func() {
				s.pahoClient.On("IsConnected").Return(true)
				s.pahoClient.On("Subscribe", testTopic, byte(1), mock.Anything).
					Return(timeoutToken()).Once()
			},
			ExpErr: m.ErrSubscribeFailed,
		},
		{
			Case:      "pahoClient nil (Connect never called) → ErrNotConnected",
			Topic:     testTopic,
			Connected: false,
			MockFunc:  func() {},
			ExpErr:    m.ErrNotConnected,
		},
		{
			Case:      "pahoClient present but IsConnected() false → ErrNotConnected",
			Topic:     testTopic,
			Connected: true,
			MockFunc: func() {
				s.pahoClient.On("IsConnected").Return(false)
			},
			ExpErr: m.ErrNotConnected,
		},
	}

	for idx, c := range testcases {
		s.SetupTest()
		i := s.buildImpl()
		c.MockFunc()

		if c.Connected {
			i.pahoClient = s.pahoClient
		}

		err := i.Subscribe(mockCTX, c.Topic, noopHandler)
		s.Equal(c.ExpErr, err, fmt.Sprintf(errMsgUnexpected, idx, c.Case))
		s.pahoClient.AssertExpectations(s.T())
	}
}

// ─── Test: Subscribe – message handler is invoked with correct arguments ──────

func (s *mqttTestSuite) TestSubscribeHandlerInvoked() {
	i := s.buildImpl()
	i.pahoClient = s.pahoClient

	s.pahoClient.On("IsConnected").Return(true)

	// Capture the Paho MessageHandler that impl passes to Subscribe.
	var registeredHandler pahomqtt.MessageHandler
	s.pahoClient.On("Subscribe", testTopic, byte(1), mock.MatchedBy(func(h pahomqtt.MessageHandler) bool {
		registeredHandler = h
		return true
	})).Return(okToken()).Once()

	var capturedTopic string
	var capturedPayload []byte

	userHandler := func(_ ctx.CTX, topic string, payload []byte) m.MQTTError {
		capturedTopic = topic
		capturedPayload = payload
		return nil
	}

	err := i.Subscribe(mockCTX, testTopic, userHandler)
	s.Nil(err)
	s.Require().NotNil(registeredHandler, "impl must register a non-nil Paho MessageHandler")

	// Simulate the broker delivering a message.
	msg := &mockMessage{topic: testTopic, payload: []byte(`{"val":7}`)}
	registeredHandler(nil, msg)

	s.Equal(testTopic, capturedTopic)
	s.Equal([]byte(`{"val":7}`), capturedPayload)
	s.pahoClient.AssertExpectations(s.T())
}

// ─── Test: Subscribe – handler error is absorbed (only logged, not propagated) ─

func (s *mqttTestSuite) TestSubscribeHandlerErrorAbsorbed() {
	i := s.buildImpl()
	i.pahoClient = s.pahoClient

	s.pahoClient.On("IsConnected").Return(true)

	var registeredHandler pahomqtt.MessageHandler
	s.pahoClient.On("Subscribe", testTopic, byte(1), mock.MatchedBy(func(h pahomqtt.MessageHandler) bool {
		registeredHandler = h
		return true
	})).Return(okToken()).Once()

	errHandler := func(_ ctx.CTX, _ string, _ []byte) m.MQTTError {
		return errors.New("processing failed")
	}

	err := i.Subscribe(mockCTX, testTopic, errHandler)
	s.Nil(err)
	s.Require().NotNil(registeredHandler)

	// registeredHandler must not panic when the user handler returns an error.
	s.NotPanics(func() {
		registeredHandler(nil, &mockMessage{topic: testTopic, payload: []byte("x")})
	})
}

// ─── Test: buildSignedWSSURL ──────────────────────────────────────────────────

// TestBuildSignedWSSURL verifies the structural correctness of the generated
// pre-signed WebSocket URL without validating the exact signature value (which
// would require mocking time.Now).
func (s *mqttTestSuite) TestBuildSignedWSSURL() {
	i := &impl{
		conf: defaultConf(),
		awsCfg: aws.Config{
			Credentials: awsCredentials.NewStaticCredentialsProvider(
				"AKIAIOSFODNN7EXAMPLE",
				"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				"",
			),
		},
	}

	rawURL, err := i.buildSignedWSSURL(context.Background())
	s.Require().NoError(err)
	s.Require().NotEmpty(rawURL)

	parsed, parseErr := url.Parse(rawURL)
	s.Require().NoError(parseErr)

	s.Equal("wss", parsed.Scheme, "scheme must be wss")
	s.Equal(testEndpoint, parsed.Host, "host must match configured endpoint")
	s.Equal("/mqtt", parsed.Path, "path must be /mqtt")

	q := parsed.Query()
	s.Equal("AWS4-HMAC-SHA256", q.Get("X-Amz-Algorithm"))
	s.Contains(q.Get("X-Amz-Credential"), "AKIAIOSFODNN7EXAMPLE")
	s.NotEmpty(q.Get("X-Amz-Date"), "X-Amz-Date must be present")
	s.Equal("host", q.Get("X-Amz-SignedHeaders"))
	s.NotEmpty(q.Get("X-Amz-Signature"), "X-Amz-Signature must be present")
	// No session token for static (non-STS) credentials.
	s.Empty(q.Get("X-Amz-Security-Token"), "no security token for static creds")
}

// TestBuildSignedWSSURLWithSessionToken verifies that a session token is
// included in the pre-signed URL when the resolved credentials carry one.
func (s *mqttTestSuite) TestBuildSignedWSSURLWithSessionToken() {
	i := &impl{
		conf: defaultConf(),
		awsCfg: aws.Config{
			Credentials: awsCredentials.NewStaticCredentialsProvider("AKID", "SECRET", "MY-SESSION-TOKEN"),
		},
	}

	rawURL, err := i.buildSignedWSSURL(context.Background())
	s.Require().NoError(err)

	parsed, parseErr := url.Parse(rawURL)
	s.Require().NoError(parseErr)
	s.Equal("MY-SESSION-TOKEN", parsed.Query().Get("X-Amz-Security-Token"))
}

// TestBuildSignedWSSURLCredentialScopeFormat verifies that X-Amz-Credential
// embeds the expected scope sub-components (datestamp / region / service).
func (s *mqttTestSuite) TestBuildSignedWSSURLCredentialScopeFormat() {
	i := &impl{
		conf: defaultConf(),
		awsCfg: aws.Config{
			Credentials: awsCredentials.NewStaticCredentialsProvider("AKID", "SECRET", ""),
		},
	}

	rawURL, err := i.buildSignedWSSURL(context.Background())
	s.Require().NoError(err)

	parsed, _ := url.Parse(rawURL)
	credential := parsed.Query().Get("X-Amz-Credential")

	s.Contains(credential, testRegion, "credential scope must contain the region")
	s.Contains(credential, "iotdevicegateway", "credential scope must contain the service name")
	s.Contains(credential, "aws4_request", "credential scope must end with aws4_request")
}
