package kv

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPAPI(t *testing.T) {
	_, servers := makeKV(3)
	leader := findLeader(t, servers)
	ts := httptest.NewServer(servers[leader].Handler())
	defer ts.Close()

	do := func(base, method, path, body string) (int, string) {
		req, err := http.NewRequest(method, base+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	if code, _ := do(ts.URL, "PUT", "/kv/a", "1"); code != http.StatusNoContent {
		t.Fatalf("put: status %d", code)
	}
	if code, _ := do(ts.URL, "POST", "/kv/a", "2"); code != http.StatusNoContent {
		t.Fatalf("append: status %d", code)
	}
	if code, body := do(ts.URL, "GET", "/kv/a", ""); code != http.StatusOK || body != "12" {
		t.Fatalf("get: status %d body %q", code, body)
	}
	if code, _ := do(ts.URL, "DELETE", "/kv/a", ""); code != http.StatusNoContent {
		t.Fatalf("delete: status %d", code)
	}
	if code, _ := do(ts.URL, "GET", "/kv/a", ""); code != http.StatusNotFound {
		t.Fatalf("get deleted: status %d", code)
	}

	fts := httptest.NewServer(servers[(leader+1)%3].Handler())
	defer fts.Close()
	if code, _ := do(fts.URL, "PUT", "/kv/x", "1"); code != http.StatusServiceUnavailable {
		t.Fatalf("follower put: status %d, want 503", code)
	}
}