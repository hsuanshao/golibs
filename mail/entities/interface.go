package entities

import "github.com/hsuanshao/golibs/ctx"

type MailClient interface {
	Send(ctx ctx.CTX, mail *Mail) MailError
}
