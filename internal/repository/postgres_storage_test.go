package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

func TestIsPgConnRetriable_TrueForClass08Codes(t *testing.T) {
	class08Codes := []string{
		pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.SQLClientUnableToEstablishSQLConnection,
		pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
		pgerrcode.TransactionResolutionUnknown,
		pgerrcode.ProtocolViolation,
	}

	for _, code := range class08Codes {
		t.Run(code, func(t *testing.T) {
			err := fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: code})
			if !isPgConnRetriable(err) {
				t.Fatalf("isPgConnRetriable(code=%s) = false, want true", code)
			}
		})
	}
}

func TestIsPgConnRetriable_FalseForOtherCodes(t *testing.T) {
	// UniqueViolation — тот самый пример из задания: класс 23, не 08, повторять не нужно.
	err := &pgconn.PgError{Code: pgerrcode.UniqueViolation}
	if isPgConnRetriable(err) {
		t.Fatalf("isPgConnRetriable(UniqueViolation) = true, want false")
	}
}

func TestIsPgConnRetriable_FalseForNonPgError(t *testing.T) {
	if isPgConnRetriable(errors.New("plain error")) {
		t.Fatal("expected false for a non-pgconn.PgError error")
	}
	if isPgConnRetriable(nil) {
		t.Fatal("expected false for nil error")
	}
}

func newTestPostgresStorage(t *testing.T) *PostgresStorage {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN не задан — пропускаем интеграционный тест PostgresStorage")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	s := NewPostgresStorage(db, zap.NewNop())

	t.Cleanup(func() {
		db.Exec("DELETE FROM gauges WHERE id LIKE 'test_%'")
		db.Exec("DELETE FROM counters WHERE id LIKE 'test_%'")
	})

	return s
}

func TestPostgresStorage_UpdateGaugeAndCounter(t *testing.T) {
	s := newTestPostgresStorage(t)

	if err := s.UpdateGauge("test_Alloc", 10.5); err != nil {
		t.Fatalf("UpdateGauge: %v", err)
	}
	v, ok := s.Gauge("test_Alloc")
	if !ok || v != 10.5 {
		t.Fatalf("Gauge(test_Alloc) = %v, %v; want 10.5, true", v, ok)
	}

	// повторная запись должна заменять значение, а не складывать
	if err := s.UpdateGauge("test_Alloc", 20); err != nil {
		t.Fatalf("UpdateGauge: %v", err)
	}
	v, ok = s.Gauge("test_Alloc")
	if !ok || v != 20 {
		t.Fatalf("Gauge(test_Alloc) after overwrite = %v, %v; want 20, true", v, ok)
	}

	if err := s.UpdateCounter("test_PollCount", 1); err != nil {
		t.Fatalf("UpdateCounter: %v", err)
	}
	if err := s.UpdateCounter("test_PollCount", 2); err != nil {
		t.Fatalf("UpdateCounter: %v", err)
	}
	c, ok := s.Counter("test_PollCount")
	if !ok || c != 3 {
		t.Fatalf("Counter(test_PollCount) = %v, %v; want 3, true", c, ok)
	}
}

func TestPostgresStorage_MissingKey(t *testing.T) {
	s := newTestPostgresStorage(t)

	if _, ok := s.Gauge("test_missing"); ok {
		t.Fatal("Gauge(test_missing) ok = true; want false")
	}
	if _, ok := s.Counter("test_missing"); ok {
		t.Fatal("Counter(test_missing) ok = true; want false")
	}
}

func TestPostgresStorage_AllGaugesAndCounters(t *testing.T) {
	s := newTestPostgresStorage(t)

	if err := s.UpdateGauge("test_Alloc", 1.5); err != nil {
		t.Fatalf("UpdateGauge: %v", err)
	}
	if err := s.UpdateCounter("test_PollCount", 5); err != nil {
		t.Fatalf("UpdateCounter: %v", err)
	}

	gauges := s.AllGauges()
	if v, ok := gauges["test_Alloc"]; !ok || v != 1.5 {
		t.Fatalf("AllGauges()[test_Alloc] = %v, %v; want 1.5, true", v, ok)
	}

	counters := s.AllCounters()
	if v, ok := counters["test_PollCount"]; !ok || v != 5 {
		t.Fatalf("AllCounters()[test_PollCount] = %v, %v; want 5, true", v, ok)
	}
}

func TestPostgresStorage_UpdateBatch(t *testing.T) {
	s := newTestPostgresStorage(t)

	gaugeValue := 42.5
	firstDelta := int64(1)
	secondDelta := int64(2)

	err := s.UpdateBatch([]models.Metrics{
		{ID: "test_Alloc", MType: models.Gauge, Value: &gaugeValue},
		{ID: "test_PollCount", MType: models.Counter, Delta: &firstDelta},
		{ID: "test_PollCount", MType: models.Counter, Delta: &secondDelta},
		{ID: "test_BadGauge", MType: models.Gauge, Value: nil},
		{ID: "test_BadCounter", MType: models.Counter, Delta: nil},
	})
	if err != nil {
		t.Fatalf("UpdateBatch: %v", err)
	}

	v, ok := s.Gauge("test_Alloc")
	if !ok || v != 42.5 {
		t.Fatalf("Gauge(test_Alloc) = %v, %v; want 42.5, true", v, ok)
	}

	c, ok := s.Counter("test_PollCount")
	if !ok || c != 3 {
		t.Fatalf("Counter(test_PollCount) = %v, %v; want 3, true", c, ok)
	}

	if _, ok := s.Gauge("test_BadGauge"); ok {
		t.Fatal("Gauge(test_BadGauge) ok = true; want false (nil Value must be skipped)")
	}
	if _, ok := s.Counter("test_BadCounter"); ok {
		t.Fatal("Counter(test_BadCounter) ok = true; want false (nil Delta must be skipped)")
	}
}
