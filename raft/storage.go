package raft

import (
	"os"
	"sync"
)

type Storage interface {
	Save(data []byte) error
	Load() ([]byte, error)
}

type MemStorage struct {
	mu   sync.Mutex
	data []byte
}

func NewMemStorage() *MemStorage {
	return &MemStorage{}
}

func (m *MemStorage) Save(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = append([]byte(nil), data...)
	return nil
}

func (m *MemStorage) Load() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]byte(nil), m.data...), nil
}

type FileStorage struct {
	mu   sync.Mutex
	path string
}

func NewFileStorage(path string) *FileStorage {
	return &FileStorage{path: path}
}

func (f *FileStorage) Save(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	tmp := f.path + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

func (f *FileStorage) Load() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}