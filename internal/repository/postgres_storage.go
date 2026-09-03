package repository

import (
	"database/sql"

	"go.uber.org/zap"
)

type PostgresStorage struct {
	db  *sql.DB
	log *zap.Logger
}

func NewPostgresStorage(db *sql.DB, log *zap.Logger) *PostgresStorage {
	return &PostgresStorage{db: db, log: log}
}

func (s PostgresStorage) UpdateGauge(name string, value float64) error {
	_, err := s.db.Exec("INSERT INTO gauges (id, value) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value", name, value)
	if err != nil {
		s.log.Error("Insert gauge failed", zap.Error(err))
		return err
	}
	return nil
}

func (s PostgresStorage) UpdateCounter(name string, value int64) error {
	_, err := s.db.Exec("INSERT INTO counters (id, delta) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET delta = counters.delta + EXCLUDED.delta", name, value)
	if err != nil {
		s.log.Error("Insert counter failed", zap.Error(err))
		return err
	}
	return nil
}

func (s PostgresStorage) Gauge(name string) (float64, bool) {
	var value float64
	err := s.db.QueryRow("SELECT value FROM gauges WHERE id = $1", name).Scan(&value)
	if err != nil {
		return 0, false
	}
	return value, true

}

func (s PostgresStorage) Counter(name string) (int64, bool) {
	var delta int64
	err := s.db.QueryRow("SELECT delta FROM counters WHERE id = $1", name).Scan(&delta)
	if err != nil {
		return 0, false
	}
	return delta, true
}

func (s PostgresStorage) AllGauges() map[string]float64 {
	gauges := make(map[string]float64)

	rows, err := s.db.Query("SELECT id, value FROM gauges")
	if err != nil {
		s.log.Error("Query all gauges failed", zap.Error(err))
		return gauges
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var value float64
		if err := rows.Scan(&id, &value); err != nil {
			s.log.Error("Scan all gauges failed", zap.Error(err))
			return gauges
		}
		gauges[id] = value
	}
	return gauges
}

func (s PostgresStorage) AllCounters() map[string]int64 {
	counters := make(map[string]int64)
	rows, err := s.db.Query("SELECT id, delta FROM counters")
	if err != nil {
		s.log.Error("Query all counters failed", zap.Error(err))
		return counters
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var delta int64
		if err := rows.Scan(&id, &delta); err != nil {
			s.log.Error("Scan all counters failed", zap.Error(err))
			return counters
		}
		counters[id] = delta
	}
	return counters
}
