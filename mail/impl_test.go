package mail

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsSes "github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/hsuanshao/golibs/ctx"
	m "github.com/hsuanshao/golibs/mail/entities"
)

// ─── Mock ─────────────────────────────────────────────────────────────────────

// MockSESClient mocks SESClientAPI so tests never reach the network.
type MockSESClient struct {
	mock.Mock
}

func (mc *MockSESClient) SendEmail(c context.Context, params *awsSes.SendEmailInput, optFns ...func(*awsSes.Options)) (*awsSes.SendEmailOutput, error) {
	args := mc.Called(c, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*awsSes.SendEmailOutput), args.Error(1)
}

// ─── Constants ───────────────────────────────────────────────────────────────

const (
	testSender          = "sender@example.com"
	testRecipient       = "to@example.com"
	testRegionUS        = "us-east-1"
	testRegionAP        = "ap-northeast-1"
	testAccessKeyID     = "AKIAIOSFODNN7EXAMPLE"
	testSecretAccessKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	errMsgUnexpected    = "Case[%d] %s: unexpected error"
	errMsgClientNil     = "Case[%d] %s: client nil-ness mismatch"
)

// ─── Suite ────────────────────────────────────────────────────────────────────

var (
	mockCTX    = ctx.Background()
	defaultSES = new(MockSESClient)
)

type mailTestSuite struct {
	suite.Suite
	client m.MailClient
}

func TestMailSuite(t *testing.T) {
	ctx.SetDebugLevel()
	suite.Run(t, new(mailTestSuite))
}

func (s *mailTestSuite) SetupTest() {
	mockCTX = ctx.Background()
	defaultSES = new(MockSESClient)
	s.client = &impl{ses: defaultSES}
}

// ─── Test: Send ───────────────────────────────────────────────────────────────

func (s *mailTestSuite) TestSend() {
	testcases := []struct {
		Case     string
		MockFunc func()
		Mail     *m.Mail
		ExpErr   error
	}{
		{
			Case: "normal case: to + cc + bcc",
			MockFunc: func() {
				defaultSES.On("SendEmail", mock.Anything, mock.MatchedBy(func(input *awsSes.SendEmailInput) bool {
					return *input.FromEmailAddress == testSender &&
						len(input.Destination.ToAddresses) == 1 &&
						len(input.Destination.CcAddresses) == 1 &&
						len(input.Destination.BccAddresses) == 1
				}), mock.Anything).Return(&awsSes.SendEmailOutput{
					MessageId: aws.String("mock-message-id-001"),
				}, nil).Once()
			},
			Mail: &m.Mail{
				From:    testSender,
				To:      []string{testRecipient},
				Cc:      []string{"cc@example.com"},
				Bcc:     []string{"bcc@example.com"},
				Subject: "Hello",
				Body:    "<p>Hi there</p>",
			},
			ExpErr: nil,
		},
		{
			Case: "normal case: to only, no cc/bcc",
			MockFunc: func() {
				defaultSES.On("SendEmail", mock.Anything, mock.MatchedBy(func(input *awsSes.SendEmailInput) bool {
					return *input.FromEmailAddress == testSender &&
						len(input.Destination.ToAddresses) == 2 &&
						input.Destination.CcAddresses == nil &&
						input.Destination.BccAddresses == nil
				}), mock.Anything).Return(&awsSes.SendEmailOutput{
					MessageId: aws.String("mock-message-id-002"),
				}, nil).Once()
			},
			Mail: &m.Mail{
				From:    testSender,
				To:      []string{"a@example.com", "b@example.com"},
				Subject: "Batch",
				Body:    "<p>Batch mail</p>",
			},
			ExpErr: nil,
		},
		{
			Case: "nil mail returns ErrSendMailFailed",
			MockFunc: func() {
				// validateMail rejects before any SES interaction.
			},
			Mail:   nil,
			ExpErr: m.ErrSendMailFailed,
		},
		{
			Case: "empty From returns ErrSendMailFailed",
			MockFunc: func() {
				// validateMail rejects before any SES interaction.
			},
			Mail: &m.Mail{
				From:    "",
				To:      []string{testRecipient},
				Subject: "Test",
				Body:    "body",
			},
			ExpErr: m.ErrSendMailFailed,
		},
		{
			Case: "empty To returns ErrSendMailFailed",
			MockFunc: func() {
				// validateMail rejects before any SES interaction.
			},
			Mail: &m.Mail{
				From:    testSender,
				To:      []string{},
				Subject: "Test",
				Body:    "body",
			},
			ExpErr: m.ErrSendMailFailed,
		},
		{
			Case: "SES SendEmail API error returns ErrSendMailFailed",
			MockFunc: func() {
				defaultSES.On("SendEmail", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("MessageRejected: Email address is not verified")).Once()
			},
			Mail: &m.Mail{
				From:    testSender,
				To:      []string{testRecipient},
				Subject: "Test",
				Body:    "body",
			},
			ExpErr: m.ErrSendMailFailed,
		},
		{
			Case: "SES returns output with nil MessageId (still success)",
			MockFunc: func() {
				defaultSES.On("SendEmail", mock.Anything, mock.Anything, mock.Anything).
					Return(&awsSes.SendEmailOutput{MessageId: nil}, nil).Once()
			},
			Mail: &m.Mail{
				From:    testSender,
				To:      []string{testRecipient},
				Subject: "No-ID",
				Body:    "body",
			},
			ExpErr: nil,
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case_no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case_name", c.Case)

		c.MockFunc()
		err := s.client.Send(mockCTX, c.Mail)
		s.Equal(c.ExpErr, err, fmt.Sprintf(errMsgUnexpected, idx, c.Case))

		// Reset mocks for the next iteration.
		defaultSES = new(MockSESClient)
		s.client = &impl{ses: defaultSES}
	}
}

// ─── Test: validateConfig ─────────────────────────────────────────────────────

func (s *mailTestSuite) TestValidateConfig() {
	testcases := []struct {
		Case   string
		Conf   *m.Config
		ExpErr error
	}{
		{
			Case:   "nil config",
			Conf:   nil,
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "empty region",
			Conf:   &m.Config{Region: ""},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "blank region (spaces only)",
			Conf:   &m.Config{Region: "   "},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			// Option is provided but RoleARN is nil → misconfiguration.
			Case:   "Option present, RoleARN nil → ErrInvalidConfig",
			Conf:   &m.Config{Region: testRegionAP, Option: &m.ConfigOption{RoleARN: nil}},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			// Option is provided but RoleARN is empty string → misconfiguration.
			Case:   "Option present, RoleARN empty string → ErrInvalidConfig",
			Conf:   &m.Config{Region: testRegionAP, Option: &m.ConfigOption{RoleARN: aws.String("")}},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			// Option is provided but RoleARN is blank (spaces) → misconfiguration.
			Case:   "Option present, RoleARN blank string → ErrInvalidConfig",
			Conf:   &m.Config{Region: testRegionAP, Option: &m.ConfigOption{RoleARN: aws.String("   ")}},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case:   "valid config, no Option → ErrInvalidConfig",
			Conf:   &m.Config{Region: testRegionUS},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case: "valid config with non-empty RoleARN",
			Conf: &m.Config{
				Region: testRegionAP,
				Option: &m.ConfigOption{RoleARN: aws.String("arn:aws:iam::123456789012:role/MailRole")},
			},
			ExpErr: nil,
		},
		{
			Case: "valid config with AccessKeyID and SecretAccessKey",
			Conf: &m.Config{
				Region: testRegionUS,
				Option: &m.ConfigOption{
					AccessKeyID:     aws.String(testAccessKeyID),
					SecretAccessKey: aws.String(testSecretAccessKey),
				},
			},
			ExpErr: nil,
		},
		{
			Case: "AccessKeyID only (no SecretAccessKey) → ErrInvalidConfig",
			Conf: &m.Config{
				Region: testRegionUS,
				Option: &m.ConfigOption{
					AccessKeyID: aws.String(testAccessKeyID),
				},
			},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case: "SecretAccessKey only (no AccessKeyID) → ErrInvalidConfig",
			Conf: &m.Config{
				Region: testRegionUS,
				Option: &m.ConfigOption{
					SecretAccessKey: aws.String(testSecretAccessKey),
				},
			},
			ExpErr: m.ErrInvalidConfig,
		},
		{
			Case: "both RoleARN and static keys provided → valid (RoleARN takes precedence)",
			Conf: &m.Config{
				Region: testRegionAP,
				Option: &m.ConfigOption{
					RoleARN:         aws.String("arn:aws:iam::123456789012:role/MailRole"),
					AccessKeyID:     aws.String(testAccessKeyID),
					SecretAccessKey: aws.String(testSecretAccessKey),
				},
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
// Note: config.LoadDefaultConfig and sesv2.NewFromConfig only build in-memory
// structs — they do not make any network calls. AWS credential resolution and
// STS AssumeRole are lazy-evaluated on the first actual API call, so these
// tests are safe to run without real AWS credentials.
func (s *mailTestSuite) TestNew() {
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
			Case:         "empty region → ErrInvalidConfig",
			Conf:         &m.Config{Region: ""},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			Case:         "blank region → ErrInvalidConfig",
			Conf:         &m.Config{Region: "   "},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			// Option provided but RoleARN nil → misconfiguration.
			Case:         "Option present, RoleARN nil → ErrInvalidConfig",
			Conf:         &m.Config{Region: testRegionAP, Option: &m.ConfigOption{RoleARN: nil}},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			// Option provided but RoleARN empty → misconfiguration.
			Case:         "Option present, RoleARN empty string → ErrInvalidConfig",
			Conf:         &m.Config{Region: testRegionAP, Option: &m.ConfigOption{RoleARN: aws.String("")}},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			// Option provided but RoleARN blank → misconfiguration.
			Case:         "Option present, RoleARN blank string → ErrInvalidConfig",
			Conf:         &m.Config{Region: testRegionAP, Option: &m.ConfigOption{RoleARN: aws.String("   ")}},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			// New requires Option to be set; nil Option is treated as a
			// misconfiguration even when Region is valid.
			Case:         "valid config, no Option → ErrInvalidConfig",
			Conf:         &m.Config{Region: testRegionUS},
			ExpClientNil: true,
			ExpErr:       m.ErrInvalidConfig,
		},
		{
			// RoleARN is set → AssumeRoleProvider is wired into credential chain.
			// stscreds.NewAssumeRoleProvider and aws.NewCredentialsCache are pure
			// in-memory structs; the STS network call only happens on the first
			// real SES API invocation, so no network is touched here.
			Case: "valid config, RoleARN set → assume-role provider wired, success",
			Conf: &m.Config{
				Region: testRegionAP,
				Option: &m.ConfigOption{
					RoleARN: aws.String("arn:aws:iam::123456789012:role/SESMailRole"),
				},
			},
			ExpClientNil: false,
			ExpErr:       nil,
		},
		{
			// Static credentials → credentials.NewStaticCredentialsProvider is
			// a pure in-memory struct, no network call.
			Case: "valid config, static keys → static credential provider wired, success",
			Conf: &m.Config{
				Region: testRegionUS,
				Option: &m.ConfigOption{
					AccessKeyID:     aws.String(testAccessKeyID),
					SecretAccessKey: aws.String(testSecretAccessKey),
				},
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
