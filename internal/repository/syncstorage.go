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

func (s *SyncStorage) UpdateGauge(name string, value float64) error {
	if err := s.Storage.UpdateGauge(name, value); err != nil {
		return err
	}
	return SaveToFile(s.Storage, s.path)
}

func (s *SyncStorage) UpdateCounter(name string, value int64) error {
	if err := s.Storage.UpdateCounter(name, value); err != nil {
		return err
	}
	return SaveToFile(s.Storage, s.path)
}
