package kv

func (s *Store) Append(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] += value
}