package qpdf

import (
	"github.com/hsuanshao/golibs/ctx"
)

/**
 * interface describe opdf's functions
 * opdf is based on
 * https://github.com/johnfercher/maroto
 *
 * for friendly call
 */

type PDF interface {
	// Initial to Initial a pdf
	Initial(ctx ctx.CTX, orientation Orientation, pageSize PageSize) (err error)
	// SetHeader to set document header
	SetHeader(ctx ctx.CTX, header *HeaderTable) (err error)
	// SetFooter to set document footer
	SetFooter(ctx ctx.CTX, footer *FooterTable) (err error)
	// AppendAbstractTable to append an abstrct table on current page
	AppendAbstractTable(ctx ctx.CTX, table *AbstractTable) (err error)
	// AppendTable to append a table collection on current page
	AppendTable(ctx ctx.CTX, table *TableForm) (err error)
	// AddPage to add a new page in PDF
	AddPage(ctx ctx.CTX) (err error)
	// GetPDFbyte to get pdf content (byte)
	GetPDFbyte(ctx ctx.CTX) (pdfContent []byte, err error)
	// Save to close pdf file, and save in temp location
	Save(ctx ctx.CTX, filename string) (tempPath string, err error)
}
