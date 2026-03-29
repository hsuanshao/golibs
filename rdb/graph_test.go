package rdb

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/hsuanshao/golibs/ctx"
)

// ---------------------------------------------------------------------------
// mockDBClient — a lightweight in-package mock for DBClient,
// avoiding circular imports with the generated mocks/ subpackage.
// ---------------------------------------------------------------------------

type mockDBClientForGraph struct {
	mock.Mock
}

func (m *mockDBClientForGraph) Exec(c ctx.CTX, sql string, args ...any) (pgconn.CommandTag, DBError) {
	called := m.Called(c, sql, args)
	return called.Get(0).(pgconn.CommandTag), called.Error(1)
}

func (m *mockDBClientForGraph) Query(c ctx.CTX, sql string, args ...any) (pgx.Rows, DBError) {
	called := m.Called(c, sql, args)
	var rows pgx.Rows
	if called.Get(0) != nil {
		rows = called.Get(0).(pgx.Rows)
	}
	return rows, called.Error(1)
}

func (m *mockDBClientForGraph) QueryRow(c ctx.CTX, sql string, args ...any) pgx.Row {
	called := m.Called(c, sql, args)
	return called.Get(0).(pgx.Row)
}

func (m *mockDBClientForGraph) WithTransaction(c ctx.CTX, fn TxBlock) DBError {
	called := m.Called(c, fn)
	return called.Error(0)
}

func (m *mockDBClientForGraph) NewBatchWriter(c ctx.CTX) BatchWriter {
	called := m.Called(c)
	return called.Get(0).(BatchWriter)
}

func (m *mockDBClientForGraph) Leader() DBClient   { return m }
func (m *mockDBClientForGraph) Follower() DBClient { return m }
func (m *mockDBClientForGraph) Stats(c ctx.CTX) DBStats {
	return DBStats{}
}
func (m *mockDBClientForGraph) Close() {}

// ---------------------------------------------------------------------------
// graphTestSuite groups all GraphClient unit tests using testify/suite.
// ---------------------------------------------------------------------------

type graphTestSuite struct {
	suite.Suite
	mockDB      *mockDBClientForGraph
	graphClient GraphClient
	ctx         ctx.CTX
}

func TestGraphClient(t *testing.T) {
	ctx.SetDebugLevel()
	suite.Run(t, new(graphTestSuite))
}

func (s *graphTestSuite) SetupTest() {
	s.mockDB = &mockDBClientForGraph{}
	s.mockDB.Test(s.T())
	s.graphClient = NewGraphClient(s.mockDB)
	s.ctx = ctx.Background()
}

// ---------------------------------------------------------------------------
// GraphError tests
// ---------------------------------------------------------------------------

func (s *graphTestSuite) TestGraphErrorWithUnderlying() {
	underlying := errors.New("connection reset")
	ge := &GraphError{Op: "ExecuteCypher", Graph: "test_graph", Message: "query failed", Err: underlying}
	s.Contains(ge.Error(), "graph(test_graph)")
	s.Contains(ge.Error(), "ExecuteCypher")
	s.Contains(ge.Error(), "query failed")
	s.Contains(ge.Error(), "connection reset")
	s.True(errors.Is(ge, underlying))
}

func (s *graphTestSuite) TestGraphErrorWithoutUnderlying() {
	ge := &GraphError{Op: "MutateCypher", Graph: "g", Message: "empty query"}
	s.Contains(ge.Error(), "graph(g)")
	s.Contains(ge.Error(), "MutateCypher")
	s.Contains(ge.Error(), "empty query")
	s.NotContains(ge.Error(), "<nil>")
}

// ---------------------------------------------------------------------------
// ParseAgtype tests
// ---------------------------------------------------------------------------

func (s *graphTestSuite) TestParseAgtypeVertex() {
	raw := `{"id": 1125899906842625, "label": "Panel", "properties": {"sn": "ABC123", "device_id": "DEV001"}}::vertex`
	result, err := ParseAgtype(raw)
	s.NoError(err)

	v, ok := result.(Vertex)
	s.True(ok)
	s.Equal(int64(1125899906842625), v.ID)
	s.Equal("Panel", v.Label)
	s.Equal("ABC123", v.Properties["sn"])
	s.Equal("DEV001", v.Properties["device_id"])
}

func (s *graphTestSuite) TestParseAgtypeEdge() {
	raw := `{"id": 2251799813685249, "label": "RUNS_FIRMWARE", "start_id": 1125899906842625, "end_id": 3377699720527873, "properties": {"since": "2025-01-15"}}::edge`
	result, err := ParseAgtype(raw)
	s.NoError(err)

	e, ok := result.(Edge)
	s.True(ok)
	s.Equal(int64(2251799813685249), e.ID)
	s.Equal("RUNS_FIRMWARE", e.Label)
	s.Equal(int64(1125899906842625), e.StartID)
	s.Equal(int64(3377699720527873), e.EndID)
	s.Equal("2025-01-15", e.Properties["since"])
}

func (s *graphTestSuite) TestParseAgtypePath() {
	raw := `[{"id": 1, "label": "Panel", "properties": {"sn": "X"}}::vertex, {"id": 10, "label": "RUNS", "start_id": 1, "end_id": 2, "properties": {}}::edge, {"id": 2, "label": "FW", "properties": {"ver": "1.0"}}::vertex]::path`
	result, err := ParseAgtype(raw)
	s.NoError(err)

	p, ok := result.(Path)
	s.True(ok)
	s.Len(p.Vertices, 2)
	s.Len(p.Edges, 1)
	s.Equal("Panel", p.Vertices[0].Label)
	s.Equal("FW", p.Vertices[1].Label)
	s.Equal("RUNS", p.Edges[0].Label)
}

func (s *graphTestSuite) TestParseAgtypeScalarNumber() {
	result, err := ParseAgtype("42")
	s.NoError(err)
	s.Equal(float64(42), result)
}

func (s *graphTestSuite) TestParseAgtypeScalarString() {
	result, err := ParseAgtype(`"hello"`)
	s.NoError(err)
	s.Equal("hello", result)
}

func (s *graphTestSuite) TestParseAgtypeScalarBool() {
	result, err := ParseAgtype("true")
	s.NoError(err)
	s.Equal(true, result)
}

func (s *graphTestSuite) TestParseAgtypeScalarNull() {
	result, err := ParseAgtype("null")
	s.NoError(err)
	s.Nil(result)
}

func (s *graphTestSuite) TestParseAgtypeEmpty() {
	result, err := ParseAgtype("")
	s.NoError(err)
	s.Nil(result)
}

func (s *graphTestSuite) TestParseAgtypeInvalidVertex() {
	raw := `{invalid json}::vertex`
	_, err := ParseAgtype(raw)
	s.Error(err)
	s.True(errors.Is(err, ErrAgtypeParse))
}

func (s *graphTestSuite) TestParseAgtypeInvalidEdge() {
	raw := `{not valid}::edge`
	_, err := ParseAgtype(raw)
	s.Error(err)
	s.True(errors.Is(err, ErrAgtypeParse))
}

func (s *graphTestSuite) TestParseAgtypeInvalidPathFormat() {
	raw := `not-an-array::path`
	_, err := ParseAgtype(raw)
	s.Error(err)
	s.True(errors.Is(err, ErrAgtypeParse))
}

func (s *graphTestSuite) TestParseAgtypePathWithInvalidElement() {
	raw := `[{invalid}::vertex]::path`
	_, err := ParseAgtype(raw)
	s.Error(err)
	s.True(errors.Is(err, ErrAgtypeParse))
}

func (s *graphTestSuite) TestParseAgtypeScalarPlainString() {
	// AGE may return unquoted strings in some edge cases
	result, err := ParseAgtype("plain_text_value")
	s.NoError(err)
	s.Equal("plain_text_value", result)
}

// ---------------------------------------------------------------------------
// extractReturnAliases tests
// ---------------------------------------------------------------------------

func (s *graphTestSuite) TestExtractReturnAliasesSingle() {
	aliases := extractReturnAliases("MATCH (n) RETURN n")
	s.Equal([]string{"n"}, aliases)
}

func (s *graphTestSuite) TestExtractReturnAliasesMultiple() {
	aliases := extractReturnAliases("MATCH (n)-[r]->(m) RETURN n, r, m")
	s.Equal([]string{"n", "r", "m"}, aliases)
}

func (s *graphTestSuite) TestExtractReturnAliasesWithAS() {
	aliases := extractReturnAliases("MATCH (n) RETURN n AS node, n.name AS name")
	s.Equal([]string{"node", "name"}, aliases)
}

func (s *graphTestSuite) TestExtractReturnAliasesNoReturn() {
	aliases := extractReturnAliases("CREATE (n:Panel {sn: 'ABC'})")
	s.Empty(aliases)
}

func (s *graphTestSuite) TestExtractReturnAliasesCaseInsensitive() {
	aliases := extractReturnAliases("match (n) return n")
	s.Equal([]string{"n"}, aliases)
}

// ---------------------------------------------------------------------------
// buildCypherSQL tests
// ---------------------------------------------------------------------------

func (s *graphTestSuite) TestBuildCypherSQLWithAliases() {
	sql := buildCypherSQL("my_graph", "MATCH (n) RETURN n", []string{"n"})
	s.Contains(sql, "cypher('my_graph'")
	s.Contains(sql, "MATCH (n) RETURN n")
	s.Contains(sql, "n agtype")
}

func (s *graphTestSuite) TestBuildCypherSQLDefaultAlias() {
	sql := buildCypherSQL("g", "CREATE (n:Test)", nil)
	s.Contains(sql, "result agtype")
}

func (s *graphTestSuite) TestBuildCypherSQLMultipleAliases() {
	sql := buildCypherSQL("g", "MATCH (a)-[r]->(b) RETURN a, r, b", []string{"a", "r", "b"})
	s.Contains(sql, "a agtype, r agtype, b agtype")
}

// ---------------------------------------------------------------------------
// splitAgtypeElements tests
// ---------------------------------------------------------------------------

func (s *graphTestSuite) TestSplitAgtypeElementsSimple() {
	elements := splitAgtypeElements(`{"id":1}::vertex, {"id":2}::edge`)
	s.Len(elements, 2)
}

func (s *graphTestSuite) TestSplitAgtypeElementsNested() {
	elements := splitAgtypeElements(`{"id":1, "properties":{"a":1, "b":2}}::vertex, {"id":2}::edge`)
	s.Len(elements, 2)
}

func (s *graphTestSuite) TestSplitAgtypeElementsEmpty() {
	elements := splitAgtypeElements("")
	s.Empty(elements)
}

// ---------------------------------------------------------------------------
// ExecuteCypher validation tests
// ---------------------------------------------------------------------------

func (s *graphTestSuite) TestExecuteCypherEmptyGraphName() {
	_, err := s.graphClient.ExecuteCypher(s.ctx, "", "MATCH (n) RETURN n")
	s.Error(err)
	var ge *GraphError
	s.True(errors.As(err, &ge))
	s.Equal("ExecuteCypher", ge.Op)
	s.True(errors.Is(ge, ErrGraphNameEmpty))
}

func (s *graphTestSuite) TestExecuteCypherEmptyCypher() {
	_, err := s.graphClient.ExecuteCypher(s.ctx, "test_graph", "")
	s.Error(err)
	var ge *GraphError
	s.True(errors.As(err, &ge))
	s.True(errors.Is(ge, ErrCypherEmpty))
}

func (s *graphTestSuite) TestExecuteCypherQueryError() {
	expectedSQL := "SELECT * FROM cypher('g', $$ MATCH (n) RETURN n $$) AS (n agtype)"
	s.mockDB.On("Query", s.ctx, expectedSQL, []interface{}(nil)).
		Return(nil, DBErrExecuteQueryStmtFailed)

	_, err := s.graphClient.ExecuteCypher(s.ctx, "g", "MATCH (n) RETURN n")
	s.Error(err)
	var ge *GraphError
	s.True(errors.As(err, &ge))
	s.Equal("ExecuteCypher", ge.Op)
}

// ---------------------------------------------------------------------------
// MutateCypher validation tests
// ---------------------------------------------------------------------------

func (s *graphTestSuite) TestMutateCypherEmptyGraphName() {
	_, err := s.graphClient.MutateCypher(s.ctx, "", "CREATE (n:Test)")
	s.Error(err)
	var ge *GraphError
	s.True(errors.As(err, &ge))
	s.Equal("MutateCypher", ge.Op)
	s.True(errors.Is(ge, ErrGraphNameEmpty))
}

func (s *graphTestSuite) TestMutateCypherEmptyCypher() {
	_, err := s.graphClient.MutateCypher(s.ctx, "g", "")
	s.Error(err)
	var ge *GraphError
	s.True(errors.As(err, &ge))
	s.True(errors.Is(ge, ErrCypherEmpty))
}

func (s *graphTestSuite) TestMutateCypherExecError() {
	expectedSQL := "SELECT * FROM cypher('g', $$ CREATE (n:Test) $$) AS (result agtype)"
	s.mockDB.On("Exec", s.ctx, expectedSQL, []interface{}(nil)).
		Return(pgconn.CommandTag{}, DBErrExecuteQueryStmtFailed)

	_, err := s.graphClient.MutateCypher(s.ctx, "g", "CREATE (n:Test)")
	s.Error(err)
	var ge *GraphError
	s.True(errors.As(err, &ge))
	s.Equal("MutateCypher", ge.Op)
}

func (s *graphTestSuite) TestMutateCypherSuccess() {
	expectedSQL := "SELECT * FROM cypher('g', $$ CREATE (n:Test) RETURN n $$) AS (n agtype)"
	tag := pgconn.NewCommandTag("SELECT 3")
	s.mockDB.On("Exec", s.ctx, expectedSQL, []interface{}(nil)).
		Return(tag, nil)

	affected, err := s.graphClient.MutateCypher(s.ctx, "g", "CREATE (n:Test) RETURN n")
	s.NoError(err)
	s.Equal(int64(3), affected)
}

// ---------------------------------------------------------------------------
// NewGraphClient constructor test
// ---------------------------------------------------------------------------

func (s *graphTestSuite) TestNewGraphClientNotNil() {
	client := NewGraphClient(s.mockDB)
	s.NotNil(client)
}
