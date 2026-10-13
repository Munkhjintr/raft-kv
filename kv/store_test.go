package kv
package kv

import (
	"fmt"
	"sync"
	"testing"
)

func TestPutGetDelete(t *testing.T) {
	s := New()

	if _, ok := s.Get("a"); ok {
		t.Fatal("expected key a to be missing")
	}

	s.Put("a", "1")
	if v, ok := s.Get("a"); !ok || v != "1" {
		t.Fatalf("got (%q, %v), want (\"1\", true)", v, ok)
	}

	s.Delete("a")
	if _, ok := s.Get("a"); ok {
		t.Fatal("expected key a to be deleted")
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := New()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("k%d", i)
			s.Put(key, "v")
			s.Get(key)
			s.Delete(key)
		}(i)
	}
	wg.Wait()
}