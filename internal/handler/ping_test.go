package handler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

type fakePingConn struct {
	err error
}

func (c *fakePingConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *fakePingConn) Close() error                        { return nil }
func (c *fakePingConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }
func (c *fakePingConn) Ping(context.Context) error          { return c.err }

type fakePingDriver struct {
	err error
}

func (d *fakePingDriver) Open(string) (driver.Conn, error) {
	return &fakePingConn{err: d.err}, nil
}

func init() {
	sql.Register("fakeping-ok", &fakePingDriver{})
	sql.Register("fakeping-fail", &fakePingDriver{err: errors.New("ping failed")})
}

func TestPingHandler_Ping_Success(t *testing.T) {
	db, err := sql.Open("fakeping-ok", "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	h := NewPingHandler(db, zap.NewNop())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	h.Ping(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestPingHandler_Ping_Failure(t *testing.T) {
	db, err := sql.Open("fakeping-fail", "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	h := NewPingHandler(db, zap.NewNop())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	h.Ping(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
