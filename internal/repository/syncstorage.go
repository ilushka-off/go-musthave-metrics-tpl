package repository

type SyncStorage struct {
	Storage
	path string
}

func NewSyncStorage(storage Storage, path string) *SyncStorage {
	return &SyncStorage{
		Storage: storage,
		path:    path,
	}
}

func (s *SyncStorage) UpdateGauge(name string, value float64) {
	s.Storage.UpdateGauge(name, value)
	SaveToFile(s.Storage, s.path)
}

func (s *SyncStorage) UpdateCounter(name string, value int64) {
	s.Storage.UpdateCounter(name, value)
	SaveToFile(s.Storage, s.path)
}
