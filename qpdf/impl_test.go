package qpdf

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hsuanshao/golibs/ctx"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestCTX() ctx.CTX {
	logger := logrus.New()
	logger.Out = io.Discard
	return ctx.CTX{
		Context:     context.Background(),
		FieldLogger: logger,
	}
}

// newInitializedImpl returns an *impl that has already been initialized with
// Landscape A4 so that tests which depend on im.pdf != nil can skip the
// boilerplate.
func newInitializedImpl(t *testing.T) *impl {
	t.Helper()
	im := &impl{}
	c := newTestCTX()
	err := im.Initial(c, Landscape, A4)
	require.NoError(t, err)
	require.NotNil(t, im.pdf)
	return im
}

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	p := New()
	assert.NotNil(t, p)
	// Underlying type must be *impl
	_, ok := p.(*impl)
	assert.True(t, ok, "New() should return *impl")
}

// ---------------------------------------------------------------------------
// Initial
// ---------------------------------------------------------------------------

func TestInitialPortrait(t *testing.T) {
	c := newTestCTX()
	im := &impl{}
	err := im.Initial(c, Portrait, A4)
	assert.NoError(t, err)
	assert.NotNil(t, im.pdf)
}

func TestInitialLandscape(t *testing.T) {
	c := newTestCTX()
	im := &impl{}
	err := im.Initial(c, Landscape, A4)
	assert.NoError(t, err)
	assert.NotNil(t, im.pdf)
}

// ---------------------------------------------------------------------------
// cellFontSize (unexported)
// ---------------------------------------------------------------------------

func TestCellFontSize(t *testing.T) {
	im := &impl{}
	tests := []struct {
		name      string
		input     string
		cellWidth uint
		expected  float64
	}{
		{
			name:      "short string normal width",
			input:     "hello",
			cellWidth: 5,
			expected:  10,
		},
		{
			name:      "long string wide cell",
			input:     "this is a very long string that is definitely longer than fifty characters total",
			cellWidth: 5,
			expected:  10,
		},
		{
			name:      "long string narrow cell (width <= 3)",
			input:     "this is a very long string that exceeds fifty characters for sure!",
			cellWidth: 3,
			expected:  8,
		},
		{
			name:      "exactly 50 chars narrow cell",
			input:     "01234567890123456789012345678901234567890123456789", // 50 chars
			cellWidth: 2,
			expected:  10, // strlen == 50, not > 50
		},
		{
			name:      "51 chars narrow cell",
			input:     "012345678901234567890123456789012345678901234567890", // 51 chars
			cellWidth: 2,
			expected:  8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := im.cellFontSize(tt.input, tt.cellWidth)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ---------------------------------------------------------------------------
// getLogo (unexported) – uses httptest to avoid real network calls
// ---------------------------------------------------------------------------

func TestGetLogoSuccess(t *testing.T) {
	c := newTestCTX()
	im := &impl{}

	// Serve a tiny 1×1 white PNG
	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(pngBytes)
	}))
	defer ts.Close()

	logoURL := ts.URL + "/test_logo.png"
	localPath, err := im.getLogo(c, logoURL)
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/test_logo.png", localPath)
	assert.Equal(t, "/tmp/test_logo.png", im.logoPath)

	// Verify the file was written
	data, err := os.ReadFile(localPath)
	assert.NoError(t, err)
	assert.Equal(t, pngBytes, data)

	// Cleanup
	os.Remove(localPath)
}

func TestGetLogoInvalidURL(t *testing.T) {
	c := newTestCTX()
	im := &impl{}

	_, err := im.getLogo(c, "http://127.0.0.1:0/nonexistent.png")
	assert.ErrorIs(t, err, ErrLoadImageByURL)
}

// ---------------------------------------------------------------------------
// SetHeader
// ---------------------------------------------------------------------------

func TestSetHeaderErrLackOfInit(t *testing.T) {
	c := newTestCTX()
	im := &impl{} // pdf == nil

	header := &HeaderTable{
		HasLogo: false,
		Texts:   []string{"Title"},
	}
	err := im.SetHeader(c, header)
	assert.ErrorIs(t, err, ErrLackOfInit)
}

func TestSetHeaderWithoutLogo(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	header := &HeaderTable{
		HasLogo: false,
		Texts:   []string{"Report Title", "Subtitle"},
	}
	err := im.SetHeader(c, header)
	assert.NoError(t, err)
}

func TestSetHeaderWithLogoSuccess(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	// Stand up a fake image server
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(pngBytes)
	}))
	defer ts.Close()

	header := &HeaderTable{
		HasLogo: true,
		LogoURL: ts.URL + "/logo.png",
		Texts:   []string{"Report Title"},
	}
	err := im.SetHeader(c, header)
	assert.NoError(t, err)

	// Cleanup
	os.Remove("/tmp/logo.png")
}

func TestSetHeaderWithLogoDefaultURL(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	// HasLogo = true but LogoURL is empty → uses defaultLogo
	// defaultLogo is "" so getLogo will fail trying to fetch ""
	header := &HeaderTable{
		HasLogo: true,
		LogoURL: "",
		Texts:   []string{"Title"},
	}
	err := im.SetHeader(c, header)
	// With defaultLogo == "", the HTTP call will fail → ErrLoadImageByURL
	assert.ErrorIs(t, err, ErrLoadImageByURL)
}

// ---------------------------------------------------------------------------
// SetFooter
// ---------------------------------------------------------------------------

func TestSetFooter(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	footer := &FooterTable{
		Texts: []string{"Page 1", "Company Name", "Date"},
	}
	err := im.SetFooter(c, footer)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// AppendAbstractTable
// ---------------------------------------------------------------------------

func TestAppendAbstractTable(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	table := &AbstractTable{
		Rows: []*AbstractRow{
			{Title: "Report", Content: "Monthly Usage Report"},
			{Title: "Date", Content: "2024/01/01"},
		},
	}
	err := im.AppendAbstractTable(c, table)
	assert.NoError(t, err)
}

func TestAppendAbstractTableEmptyRows(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	table := &AbstractTable{
		Rows: []*AbstractRow{},
	}
	err := im.AppendAbstractTable(c, table)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// AppendTable
// ---------------------------------------------------------------------------

func TestAppendTableHeaderAndDataRows(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	table := &TableForm{
		Rows: []*TableRow{
			{
				IsHeader: true,
				Cells: []*TableCell{
					{Width: 2, Align: Center, Content: "Name"},
					{Width: 2, Align: Center, Content: "Value"},
				},
			},
			{
				IsHeader: false,
				Cells: []*TableCell{
					{Width: 2, Align: Left, Content: "CPU"},
					{Width: 2, Align: Left, Content: "16 cores"},
				},
			},
		},
	}
	err := im.AppendTable(c, table)
	assert.NoError(t, err)
}

func TestAppendTableCellWithMinWidth(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	// Width=1 is the effective minimum; Width=0 would panic in GetRowHeight
	// because it divides by col.Width before AppendTable's colWidth=1 fallback.
	table := &TableForm{
		Rows: []*TableRow{
			{
				IsHeader: false,
				Cells: []*TableCell{
					{Width: 1, Align: Left, Content: "short"},
				},
			},
		},
	}
	err := im.AppendTable(c, table)
	assert.NoError(t, err)
}

func TestAppendTableLongContentInNarrowCell(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	// Width == 1 and len(content) > 12 → triggers line-break insertion logic
	table := &TableForm{
		Rows: []*TableRow{
			{
				IsHeader: false,
				Cells: []*TableCell{
					{Width: 1, Align: Left, Content: "this content is definitely longer than 12 chars"},
				},
			},
		},
	}
	err := im.AppendTable(c, table)
	assert.NoError(t, err)
}

func TestAppendTableFontBoldCell(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	table := &TableForm{
		Rows: []*TableRow{
			{
				IsHeader: false,
				Cells: []*TableCell{
					{Width: 3, Align: Left, Content: "Bold item", FontBold: true},
				},
			},
		},
	}
	err := im.AppendTable(c, table)
	assert.NoError(t, err)
}

func TestAppendTableUnderLine(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	table := &TableForm{
		Rows: []*TableRow{
			{
				IsHeader: false,
				Cells: []*TableCell{
					{Width: 3, Align: Left, Content: "Underlined", UnderLine: true},
				},
			},
		},
	}
	err := im.AppendTable(c, table)
	assert.NoError(t, err)
}

func TestAppendTableEmptyRows(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	table := &TableForm{
		Rows: []*TableRow{},
	}
	err := im.AppendTable(c, table)
	assert.NoError(t, err)
}

func TestAppendTableContentWithCarriageReturn(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	// Width == 1, len > 12, but contains \r → should skip line-break insertion
	table := &TableForm{
		Rows: []*TableRow{
			{
				IsHeader: false,
				Cells: []*TableCell{
					{Width: 1, Align: Left, Content: "already\rhas breaks in it here"},
				},
			},
		},
	}
	err := im.AppendTable(c, table)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// AddPage
// ---------------------------------------------------------------------------

func TestAddPage(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	err := im.AddPage(c)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// SetPassword
// ---------------------------------------------------------------------------

func TestSetPassword(t *testing.T) {
	im := newInitializedImpl(t)
	// SetPassword should not panic
	assert.NotPanics(t, func() {
		im.SetPassword("secret123")
	})
}

// ---------------------------------------------------------------------------
// GetPDFbyte
// ---------------------------------------------------------------------------

func TestGetPDFbyteSuccess(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	content, err := im.GetPDFbyte(c)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	// PDF files begin with %PDF
	assert.True(t, len(content) > 4)
	assert.Equal(t, "%PDF", string(content[:4]))
}

// ---------------------------------------------------------------------------
// Save
// ---------------------------------------------------------------------------

func TestSaveSuccess(t *testing.T) {
	c := newTestCTX()
	im := newInitializedImpl(t)

	tmpFile := "/tmp/qpdf_test_output.pdf"
	path, err := im.Save(c, tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, tmpFile, path)

	// Verify the file exists
	info, statErr := os.Stat(tmpFile)
	assert.NoError(t, statErr)
	assert.True(t, info.Size() > 0)

	// Cleanup
	os.Remove(tmpFile)
}

// ---------------------------------------------------------------------------
// Props: AbstractTable.LongestStr
// ---------------------------------------------------------------------------

func TestLongestStr(t *testing.T) {
	tests := []struct {
		name            string
		table           *AbstractTable
		expectedTitle   float64
		expectedContent float64
	}{
		{
			name: "basic",
			table: &AbstractTable{
				Rows: []*AbstractRow{
					{Title: "short", Content: "longer content here"},
					{Title: "a longer title", Content: "x"},
				},
			},
			expectedTitle:   14, // "a longer title"
			expectedContent: 19, // "longer content here"
		},
		{
			name: "empty rows",
			table: &AbstractTable{
				Rows: []*AbstractRow{},
			},
			expectedTitle:   0,
			expectedContent: 0,
		},
		{
			name: "single row",
			table: &AbstractTable{
				Rows: []*AbstractRow{
					{Title: "abc", Content: "defgh"},
				},
			},
			expectedTitle:   3,
			expectedContent: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			titleLen, contentLen := tt.table.LongestStr()
			assert.Equal(t, tt.expectedTitle, titleLen)
			assert.Equal(t, tt.expectedContent, contentLen)
		})
	}
}

// ---------------------------------------------------------------------------
// Props: TableRow.GetRowHeight
// ---------------------------------------------------------------------------

func TestGetRowHeight(t *testing.T) {
	tests := []struct {
		name     string
		row      *TableRow
		expected float64
	}{
		{
			name: "single cell short content",
			row: &TableRow{
				Cells: []*TableCell{
					{Width: 5, Content: "short"},
				},
			},
			// strlen = 5/5 = 1 → height = 1/2 - 2*(1/18) ≈ 0.388...
			expected: float64(1)/2 - 2*(float64(1)/18),
		},
		{
			name: "single cell medium content",
			row: &TableRow{
				Cells: []*TableCell{
					{Width: 2, Content: "medium content text here"}, // len=24, 24/2=12
				},
			},
			expected: float64(12)/2 - 2*(float64(12)/18),
		},
		{
			name: "multiple cells picks longest",
			row: &TableRow{
				Cells: []*TableCell{
					{Width: 3, Content: "abc"},            // 3/3 = 1
					{Width: 2, Content: "longer content"}, // 14/2 = 7
				},
			},
			expected: float64(7)/2 - 2*(float64(7)/18),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.row.GetRowHeight()
			assert.InDelta(t, tt.expected, got, 0.001)
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: full PDF generation workflow
// ---------------------------------------------------------------------------

func TestFullWorkflow(t *testing.T) {
	c := newTestCTX()
	pdflib := New()

	// 1. Initial
	err := pdflib.Initial(c, Landscape, A4)
	assert.NoError(t, err)

	// 2. SetFooter
	err = pdflib.SetFooter(c, &FooterTable{
		Texts: []string{"Footer Left", "Footer Right"},
	})
	assert.NoError(t, err)

	// 3. AppendAbstractTable
	err = pdflib.AppendAbstractTable(c, &AbstractTable{
		Rows: []*AbstractRow{
			{Title: "Report", Content: "Test Report"},
			{Title: "Date", Content: "2024/12/01"},
		},
	})
	assert.NoError(t, err)

	// 4. AppendTable
	err = pdflib.AppendTable(c, &TableForm{
		Rows: []*TableRow{
			{
				IsHeader: true,
				Cells: []*TableCell{
					{Width: 3, Align: Center, Content: "Column A"},
					{Width: 3, Align: Center, Content: "Column B"},
				},
			},
			{
				IsHeader: false,
				Cells: []*TableCell{
					{Width: 3, Align: Left, Content: "Row1-A"},
					{Width: 3, Align: Left, Content: "Row1-B"},
				},
			},
		},
	})
	assert.NoError(t, err)

	// 5. AddPage
	err = pdflib.AddPage(c)
	assert.NoError(t, err)

	// 6. GetPDFbyte
	content, err := pdflib.GetPDFbyte(c)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)

	// 7. Save
	tmpFile := "/tmp/qpdf_full_workflow_test.pdf"
	path, err := pdflib.Save(c, tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, tmpFile, path)

	os.Remove(tmpFile)
}
