package repository

import (
	"database/sql"
	"errors"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/retry"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

// PostgresStorage is a PostgreSQL-backed implementation of Storage. Queries
// that fail with a retriable connection error (see isPgConnRetriable) are
// retried using retry.Delays.
type PostgresStorage struct {
	db  *sql.DB
	log *zap.Logger
}

// NewPostgresStorage creates a PostgresStorage backed by db. The gauges and
// counters tables are expected to already exist; see RunMigrations.
func NewPostgresStorage(db *sql.DB, log *zap.Logger) *PostgresStorage {
	return &PostgresStorage{db: db, log: log}
}

// UpdateGauge upserts the named gauge's value.
func (s PostgresStorage) UpdateGauge(name string, value float64) error {
	return retry.Do(retry.Delays, isPgConnRetriable, func() error {
		_, err := s.db.Exec("INSERT INTO gauges (id, value) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value", name, value)
		if err != nil {
			s.log.Error("Insert gauge failed", zap.Error(err))
			return err
		}
		return nil
	})
}

// UpdateCounter adds value to the named counter's running total, inserting a
// new row if the counter does not exist yet.
func (s PostgresStorage) UpdateCounter(name string, value int64) error {
	return retry.Do(retry.Delays, isPgConnRetriable, func() error {
		_, err := s.db.Exec("INSERT INTO counters (id, delta) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET delta = counters.delta + EXCLUDED.delta", name, value)
		if err != nil {
			s.log.Error("Insert counter failed", zap.Error(err))
			return err
		}
		return nil
	})

}

// Gauge returns the current value of the named gauge, or ErrNotFound if it
// has never been set.
func (s PostgresStorage) Gauge(name string) (float64, error) {
	var value float64
	err := retry.Do(retry.Delays, isPgConnRetriable, func() error {
		return s.db.QueryRow("SELECT value FROM gauges WHERE id = $1", name).Scan(&value)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		s.log.Error("Query gauge failed", zap.Error(err))
		return 0, err
	}
	return value, nil
}

// Counter returns the current value of the named counter, or ErrNotFound if
// it has never been set.
func (s PostgresStorage) Counter(name string) (int64, error) {
	var delta int64
	err := retry.Do(retry.Delays, isPgConnRetriable, func() error {
		return s.db.QueryRow("SELECT delta FROM counters WHERE id = $1", name).Scan(&delta)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		s.log.Error("Query counter failed", zap.Error(err))
		return 0, err
	}
	return delta, nil
}

// AllGauges returns every gauge and its current value. On a query error it
// logs the error and returns an empty map.
func (s PostgresStorage) AllGauges() map[string]float64 {
	gauges := make(map[string]float64)

	err := retry.Do(retry.Delays, isPgConnRetriable, func() error {
		gauges = make(map[string]float64)

		rows, err := s.db.Query("SELECT id, value FROM gauges")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id string
			var value float64
			if err := rows.Scan(&id, &value); err != nil {
				return err
			}
			gauges[id] = value
		}
		return rows.Err()
	})
	if err != nil {
		s.log.Error("Query all gauges failed", zap.Error(err))
	}
	return gauges
}

// AllCounters returns every counter and its current value. On a query error
// it logs the error and returns an empty map.
func (s PostgresStorage) AllCounters() map[string]int64 {
	counters := make(map[string]int64)

	err := retry.Do(retry.Delays, isPgConnRetriable, func() error {
		counters = make(map[string]int64)

		rows, err := s.db.Query("SELECT id, delta FROM counters")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id string
			var delta int64
			if err := rows.Scan(&id, &delta); err != nil {
				return err
			}
			counters[id] = delta
		}
		return rows.Err()
	})
	if err != nil {
		s.log.Error("Query all counters failed", zap.Error(err))
	}
	return counters
}

// UpdateBatch applies every metric in metrics within a single transaction.
// Entries with a nil Value/Delta for their type are silently skipped.
func (s PostgresStorage) UpdateBatch(metrics []models.Metrics) error {
	return retry.Do(retry.Delays, isPgConnRetriable, func() error {
		tx, err := s.db.Begin()
		if err != nil {
			s.log.Error("Transaction error", zap.Error(err))
			return err
		}

		for _, metric := range metrics {
			switch metric.MType {
			case models.Gauge:
				if metric.Value == nil {
					continue
				}
				_, err := tx.Exec("INSERT INTO gauges (id, value) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value", metric.ID, metric.Value)
				if err != nil {
					s.log.Error("Insert gauge failed", zap.Error(err))
					tx.Rollback()
					return err
				}
			case models.Counter:
				if metric.Delta == nil {
					continue
				}
				_, err := tx.Exec("INSERT INTO counters (id, delta) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET delta = counters.delta + EXCLUDED.delta", metric.ID, metric.Delta)
				if err != nil {
					s.log.Error("Insert counter failed", zap.Error(err))
					tx.Rollback()
					return err
				}
			}
		}
		if err := tx.Commit(); err != nil {
			s.log.Error("Transaction error", zap.Error(err))
			return err
		}
		return nil
	})
}

// isPgConnRetriable reports whether err is a PostgreSQL connection-class
// error worth retrying (e.g. connection loss or protocol violation).
func isPgConnRetriable(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return false
	}

	switch pgErr.Code {
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.SQLClientUnableToEstablishSQLConnection,
		pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
		pgerrcode.TransactionResolutionUnknown,
		pgerrcode.ProtocolViolation:
		return true
	default:
		return false
	}
}
