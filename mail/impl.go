package mail

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	awsSes "github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/sirupsen/logrus"

	"github.com/hsuanshao/golibs/ctx"
	m "github.com/hsuanshao/golibs/mail/entities"
)

// SESClientAPI defines the interface for SES v2 client operations used by this package.
// Abstracting the AWS client allows for easy unit-test mocking.
type SESClientAPI interface {
	SendEmail(ctx context.Context, params *awsSes.SendEmailInput, optFns ...func(*awsSes.Options)) (*awsSes.SendEmailOutput, error)
}

// New initialises a new MailClient backed by AWS SES v2.
// It validates the configuration and sets up AWS credentials using one of:
//   - IAM Role assumption (AssumeRole) when conf.Option.RoleARN is provided.
//   - Static credentials when conf.Option.AccessKeyID and SecretAccessKey are provided.
//
// At least one authentication method must be configured.
func New(c ctx.CTX, conf *m.Config) (m.MailClient, m.MailError) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefaultConfig(c, config.WithRegion(conf.Region))
	if err != nil {
		c.WithFields(logrus.Fields{"error": err, "region": conf.Region}).Error("failed to load aws default config")
		return nil, m.ErrInvalidConfig
	}

	switch {
	case hasRoleARN(conf.Option):
		stsClient := sts.NewFromConfig(cfg)
		creds := stscreds.NewAssumeRoleProvider(stsClient, *conf.Option.RoleARN)
		cfg.Credentials = aws.NewCredentialsCache(creds)
	case hasStaticKeys(conf.Option):
		cfg.Credentials = aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(
				*conf.Option.AccessKeyID,
				*conf.Option.SecretAccessKey,
				"", // session token – not required for long-term IAM user keys
			),
		)
	}

	sesClient := awsSes.NewFromConfig(cfg)
	return &impl{ses: sesClient}, nil
}

type impl struct {
	ses SESClientAPI
}

// Send delivers an email via AWS SES v2 using the simple email content format.
// All To, Cc, and Bcc addresses must be verified in SES (or the domain must be
// verified) unless the account is out of the SES sandbox.
func (c *impl) Send(ctx ctx.CTX, mail *m.Mail) m.MailError {
	if err := validateMail(mail); err != nil {
		return err
	}

	toAddresses := toAWSAddresses(mail.To)
	ccAddresses := toAWSAddresses(mail.Cc)
	bccAddresses := toAWSAddresses(mail.Bcc)

	input := &awsSes.SendEmailInput{
		FromEmailAddress: aws.String(mail.From),
		Destination: &types.Destination{
			ToAddresses:  toAddresses,
			CcAddresses:  ccAddresses,
			BccAddresses: bccAddresses,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(mail.Subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(mail.Body),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}

	output, err := c.ses.SendEmail(ctx.Context, input)
	if err != nil {
		ctx.WithFields(logrus.Fields{
			"err":     err,
			"from":    mail.From,
			"to":      mail.To,
			"subject": mail.Subject,
		}).Error("send email via ses failed")
		return m.ErrSendMailFailed
	}

	messageID := ""
	if output.MessageId != nil {
		messageID = *output.MessageId
	}
	ctx.WithFields(logrus.Fields{
		"message_id": messageID,
		"from":       mail.From,
		"to":         mail.To,
		"subject":    mail.Subject,
	}).Info("email sent successfully via ses")

	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func validateConfig(conf *m.Config) m.MailError {
	if conf == nil {
		return m.ErrInvalidConfig
	}
	if strings.TrimSpace(conf.Region) == "" {
		return m.ErrInvalidConfig
	}
	if conf.Option == nil {
		return m.ErrInvalidConfig
	}

	role := hasRoleARN(conf.Option)
	keys := hasStaticKeys(conf.Option)

	// At least one authentication method (Role or Keys) must be provided.
	if !role && !keys {
		return m.ErrInvalidConfig
	}
	return nil
}

// hasRoleARN returns true when a non-blank RoleARN is set.
func hasRoleARN(opt *m.ConfigOption) bool {
	return opt.RoleARN != nil && strings.TrimSpace(*opt.RoleARN) != ""
}

// hasStaticKeys returns true when both AccessKeyID and SecretAccessKey are set
// and non-blank. Both fields must be present together.
func hasStaticKeys(opt *m.ConfigOption) bool {
	return opt.AccessKeyID != nil && strings.TrimSpace(*opt.AccessKeyID) != "" &&
		opt.SecretAccessKey != nil && strings.TrimSpace(*opt.SecretAccessKey) != ""
}

func validateMail(mail *m.Mail) m.MailError {
	if mail == nil {
		return m.ErrSendMailFailed
	}
	if strings.TrimSpace(mail.From) == "" {
		return m.ErrSendMailFailed
	}
	if len(mail.To) == 0 {
		return m.ErrSendMailFailed
	}
	return nil
}

// toAWSAddresses converts a Go string slice to the []string type expected by
// the SES Destination fields (no conversion needed in v2, kept for clarity).
func toAWSAddresses(addrs []string) []string {
	if len(addrs) == 0 {
		return nil
	}
	return addrs
}
