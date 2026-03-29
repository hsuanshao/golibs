package qpdf

/**
 * consts of opdf
 */

var (
	defaultLogo = ""

	defaultHeaderHeight = 15

	defaultCellGap = 2
)

/**
 * Orientation defines document orientation
 * */

// Orientation describe pdf file layout orientation
type Orientation string

const (
	Portrait  Orientation = "P"
	Landscape Orientation = "L"
)

// PageSize to define document page size
type PageSize string

const (
	A4 PageSize = "A4"
)

// Align defines text align
type Align string

const (
	Left   Align = "L"
	Right  Align = "R"
	Center Align = "C"
	Top    Align = "T"
	Bottom Align = "B"
	Middle Align = "M"
)
