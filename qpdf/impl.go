package qpdf

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/pdf"
	"github.com/johnfercher/maroto/pkg/props"
	"github.com/sirupsen/logrus"

	"github.com/hsuanshao/golibs/ctx"
)

func New() PDF {
	return &impl{}
}

var (
	defaultPwd = ""
)

type impl struct {
	pdf      *pdf.Maroto
	logoPath string
}

// Initial is the inital step  pdf
func (im *impl) Initial(ctx ctx.CTX, orientation Orientation, pageSize PageSize) (err error) {
	maroto := pdf.NewMaroto(consts.Orientation(orientation), consts.PageSize(pageSize))
	// set page margin
	maroto.SetPageMargins(5, 10, 5)

	im.pdf = &maroto
	return nil
}

func (im *impl) cellFontSize(input string, cellWidth uint) (fontsize float64) {
	strlen := len(input)
	switch {
	case strlen > 50 && cellWidth <= 3:
		fontsize = 8
	default:
		fontsize = 10
	}
	return fontsize
}

func (im *impl) getLogo(ctx ctx.CTX, logoURL string) (localTmpPath string, err error) {
	client := http.Client{}
	logoReq, err := client.Get(logoURL)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "logo url": logoURL}).Error("load logo by given url has error")
		return "", ErrLoadImageByURL
	}
	defer logoReq.Body.Close()

	logoArr := strings.Split(logoURL, "/")
	logoFilename := logoArr[len(logoArr)-1]

	logodata, err := ioutil.ReadAll(logoReq.Body)
	if err != nil {
		ctx.WithField("err", err).Error("read log image failed")
		return "", ErrImageRead
	}
	localTmpPath = "/tmp/" + logoFilename
	err = ioutil.WriteFile(localTmpPath, logodata, 0666)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "filepath": localTmpPath, "data": logodata}).Error("write logo file in tmp folder has error")
		return "", ErrImageWrite
	}
	im.logoPath = localTmpPath

	return localTmpPath, nil
}

// SetHeader to set document header
func (im *impl) SetHeader(ctx ctx.CTX, header *HeaderTable) (err error) {
	if im.pdf == nil {
		return ErrLackOfInit
	}
	pdfSrv := *im.pdf

	logoURL := header.LogoURL
	if header.HasLogo && logoURL == "" {
		logoURL = defaultLogo
	}

	if header.HasLogo {
		_, err = im.getLogo(ctx, logoURL)
		if err != nil {
			ctx.WithFields(logrus.Fields{"err": err, "logo image url": logoURL}).Error("getLogo return error")
			return ErrLoadImageByURL
		}
	}

	headerCount := len(header.Texts)
	remainWidth := 12

	//Start set header
	pdfSrv.RegisterHeader(func() {
		pdfSrv.Row(float64(defaultHeaderHeight), func() {
			if header.HasLogo {
				pdfSrv.Col(2, func() {
					err := pdfSrv.FileImage(im.logoPath, props.Rect{
						Center:  true,
						Percent: 60,
					})
					if err != nil {
						ctx.WithFields(logrus.Fields{"err": err, "image path": im.logoPath}).Error("print logo on pdf has error")
					}
				})
				remainWidth -= 2
				pdfSrv.ColSpace(uint(defaultCellGap))
			}

			avgWidth := remainWidth / 10
			for i := 0; i < headerCount; i++ {
				pdfSrv.Col(uint(remainWidth), func() {
					fs := im.cellFontSize(header.Texts[i], uint(avgWidth))
					pdfSrv.Text(header.Texts[i], props.Text{
						Size:            fs,
						Align:           consts.Left,
						VerticalPadding: 7,
					})
				})
			}

		})

	})

	return nil
}

// AppendAbstractTable to append an abstrct table on current page
func (im *impl) AppendAbstractTable(ctx ctx.CTX, table *AbstractTable) (err error) {
	pdfSrv := *im.pdf
	titlen, contentlen := table.LongestStr()
	for _, row := range table.Rows {
		pdfSrv.Row(6, func() {
			title := row.Title
			content := row.Content
			fontSize := 9

			titleCells := titlen / (titlen + contentlen) * 5
			contentCells := contentlen / (titlen + contentlen) * 6
			pdfSrv.Col(uint(titleCells), func() {
				pdfSrv.Text(title, props.Text{
					Top:   2,
					Style: consts.Bold,
					Size:  float64(fontSize),
				})
			})

			pdfSrv.Col(uint(contentCells), func() {
				pdfSrv.Text(content, props.Text{
					Top:  2,
					Size: float64(fontSize),
				})
			})
		})
	}

	pdfSrv.Line(2.0)
	return nil
}

// AppendTable to append a table collection on current page
func (im *impl) AppendTable(ctx ctx.CTX, table *TableForm) (err error) {
	var colWidth uint
	pdfSrv := *im.pdf

	for _, row := range table.Rows {
		isHeader := row.IsHeader
		rowHeight := row.GetRowHeight()

		pdfSrv.Row(rowHeight, func() {
			for _, col := range row.Cells {
				colWidth = 1
				if col.Width > 0 {
					colWidth = col.Width
				}

				pdfSrv.Col(colWidth, func() {
					fontSize := 8
					style := consts.Normal
					if isHeader || col.FontBold {
						fontSize = 8
						style = consts.Bold
					}

					cellContent := col.Content
					if colWidth == 1 {
						if !row.IsHeader && len(cellContent) > 12 && !strings.Contains(cellContent, "\r") {
							x := 0
							tmpContent := ""
							for _, c := range cellContent {
								tmpContent += string(c)
								x++
								if x == 10 {
									tmpContent += " \r\r "
									x = 0
								}
							}
							cellContent = tmpContent
						}
					}

					pdfSrv.Text(cellContent, props.Text{
						Top:             1,
						Size:            float64(fontSize),
						Style:           style,
						Extrapolate:     false,
						VerticalPadding: 1.0,
						Align:           consts.Align(col.Align),
					})

					if col.UnderLine {
						pdfSrv.Line(1.0)
					}
				})
			}
		})
		if isHeader {
			pdfSrv.Line(1.0)
		}
	}

	return nil
}

// SetFooter to set document footer
func (im *impl) SetFooter(ctx ctx.CTX, footer *FooterTable) (err error) {
	pdfSrv := *im.pdf
	pdfSrv.RegisterFooter(func() {
		pdfSrv.Row(20, func() {
			cellCount := len(footer.Texts)
			avgWidth := 12 / cellCount
			for i := 0; i < cellCount; i++ {
				pdfSrv.Col(uint(avgWidth), func() {
					pdfSrv.Text(footer.Texts[i], props.Text{
						Top:   13,
						Size:  8,
						Align: consts.Left,
					})
				})
			}
		})
	})
	return nil
}

func (im *impl) AddPage(ctx ctx.CTX) (err error) {
	pdfSrv := *im.pdf
	pdfSrv.AddPage()
	return nil
}

// SetPassword to set password protection to pdf file
func (im *impl) SetPassword(pwd string) {
	pdfSrv := *im.pdf

	pdfSrv.SetProtection(1, pwd, defaultPwd)
}

func (im *impl) GetPDFbyte(ctx ctx.CTX) (pdfContent []byte, err error) {
	pdfSrv := *im.pdf

	content, err := pdfSrv.Output()
	if err != nil {
		ctx.WithField("err", err).Error("export pdf file byte failure")
		return nil, ErrExportByte
	}
	pdfContent = content.Bytes()
	return pdfContent, nil
}

// Save to close pdf file, and save in temp location
func (im *impl) Save(ctx ctx.CTX, filename string) (tempPath string, err error) {
	tempPath = filename
	pdfSrv := *im.pdf

	err = pdfSrv.OutputFileAndClose(tempPath)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "filepath": tempPath}).Error("export PDF file failure")
		return "", errors.New("export pdf fail")
	}

	return tempPath, nil
}
