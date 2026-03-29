package qpdf

// HeaderTable decsribe header format
type HeaderTable struct {
	HasLogo bool
	LogoURL string
	Texts   []string
}

type FooterTable struct {
	Texts []string
}

// AbstractTable is special table for describe page, or document's abstraction
// format
/**
 * Abstract Table
 * ---------------------------
 * title  |  content         |  <-- AbstractRow
 * ---------------------------
 * title2 |  content         |
 * ---------------------------
 * title2 |  content         |
 * ---------------------------
 */
type AbstractTable struct {
	Height float64
	Width  float64
	Rows   []*AbstractRow
}

func (at *AbstractTable) LongestStr() (titleLen float64, contentLen float64) {
	titleLen, contentLen = 0, 0
	for _, row := range at.Rows {
		tmpTitleLen := len(row.Title)
		tmpContent := len(row.Content)

		if tmpTitleLen > int(titleLen) {
			titleLen = float64(tmpTitleLen)
		}

		if tmpContent > int(contentLen) {
			contentLen = float64(tmpContent)
		}
	}

	return titleLen, contentLen
}

// AbstractRow is
type AbstractRow struct {
	Title   string
	Content string
}

// TableForm represents table for information table
type TableForm struct {
	Rows []*TableRow
}

type TableRow struct {
	IsHeader bool
	Cells    []*TableCell
}

func (tr *TableRow) GetRowHeight() (height float64) {
	height = 10
	maxLen := 0
	for _, col := range tr.Cells {
		strlen := len(col.Content)
		strlen = strlen / int(col.Width)
		if strlen > maxLen {
			maxLen = strlen
		}
	}

	height = float64(maxLen)/2 - 2*(float64(maxLen)/18)

	return height
}

type TableCell struct {
	FontBold  bool
	UnderLine bool
	Width     uint
	Align     Align
	Content   string
}
