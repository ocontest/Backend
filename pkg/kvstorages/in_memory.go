package kvstorages

import "github.com/ocontest/backend/pkg"

type InMemoryStorage struct {
	mainStorage map[string]string
}

func NewInMemoryStorage() KVStorage {
	return InMemoryStorage{
		mainStorage: make(map[string]string),
	}
}
func (i InMemoryStorage) Save(key, value string) error {
	i.mainStorage[key] = value
	return nil
}

func (i InMemoryStorage) Get(key string) (string, error) {
	val, exists := i.mainStorage[key]
	if !exists {
		return "", pkg.ErrNotFound
	}
	return val, nil
}
