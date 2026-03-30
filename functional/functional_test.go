package functional

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	apiPkg "distributed-raft-kv-store/api"
	"distributed-raft-kv-store/kv"
	"distributed-raft-kv-store/storage"
)

type responseBody map[string]any

type testServer struct {
	server *httptest.Server
	wal    storage.WAL
}

func (s *testServer) Close() {
	if s.server != nil {
		s.server.Close()
	}
	if s.wal != nil {
		_ = s.wal.Close()
	}
}

func TestKVServiceLifecycleAndRecovery(t *testing.T) {
	root := t.TempDir()
	t.Logf("created temporary persistence directory: %s", root)

	t.Log("starting service with WAL and snapshot persistence enabled")
	server := newPersistentServer(t, root, true, true)

	t.Log("writing alpha=one")
	putJSON(t, server.server.URL+"/v1/keys/alpha", map[string]string{"value": "one"}, http.StatusOK)

	t.Log("reading alpha and verifying initial value")
	body := getJSON(t, server.server.URL+"/v1/keys/alpha", http.StatusOK)
	assertString(t, body, "value", "one")

	t.Log("updating alpha=two")
	putJSON(t, server.server.URL+"/v1/keys/alpha", map[string]string{"value": "two"}, http.StatusOK)

	t.Log("reading alpha and verifying updated value")
	body = getJSON(t, server.server.URL+"/v1/keys/alpha", http.StatusOK)
	assertString(t, body, "value", "two")

	t.Log("writing beta=three and then deleting beta")
	putJSON(t, server.server.URL+"/v1/keys/beta", map[string]string{"value": "three"}, http.StatusOK)
	deleteRequest(t, server.server.URL+"/v1/keys/beta", http.StatusNoContent)

	t.Log("verifying deleted key beta returns 404")
	getJSON(t, server.server.URL+"/v1/keys/beta", http.StatusNotFound)

	walPath := filepath.Join(root, "wal.log")
	snapshotPath := filepath.Join(root, "snapshot.json")
	t.Logf("checking persisted files: wal=%s snapshot=%s", walPath, snapshotPath)
	assertFileExists(t, walPath)
	assertFileExists(t, snapshotPath)

	t.Log("stopping service to simulate a restart")
	server.Close()

	t.Log("restarting service from the same persisted data")
	restarted := newPersistentServer(t, root, true, true)
	defer restarted.Close()

	t.Log("verifying alpha recovered with latest value")
	body = getJSON(t, restarted.server.URL+"/v1/keys/alpha", http.StatusOK)
	assertString(t, body, "value", "two")

	t.Log("verifying deleted key beta is still absent after recovery")
	getJSON(t, restarted.server.URL+"/v1/keys/beta", http.StatusNotFound)

	t.Log("writing gamma after recovery to verify service remains usable")
	putJSON(t, restarted.server.URL+"/v1/keys/gamma", map[string]string{"value": "four"}, http.StatusOK)
	body = getJSON(t, restarted.server.URL+"/v1/keys/gamma", http.StatusOK)
	assertString(t, body, "value", "four")
}

func TestKVServiceWithoutPersistenceDoesNotRecover(t *testing.T) {
	root := t.TempDir()
	t.Logf("created temporary directory for no-persistence test: %s", root)

	t.Log("starting service with WAL and snapshot disabled")
	server := newPersistentServer(t, root, false, false)

	t.Log("writing temp=ephemeral")
	putJSON(t, server.server.URL+"/v1/keys/temp", map[string]string{"value": "ephemeral"}, http.StatusOK)

	t.Log("verifying temp exists before restart")
	body := getJSON(t, server.server.URL+"/v1/keys/temp", http.StatusOK)
	assertString(t, body, "value", "ephemeral")

	t.Log("stopping service to simulate a restart")
	server.Close()

	t.Log("restarting service without persistence and verifying temp is gone")
	restarted := newPersistentServer(t, root, false, false)
	defer restarted.Close()
	getJSON(t, restarted.server.URL+"/v1/keys/temp", http.StatusNotFound)
}

func newPersistentServer(t *testing.T, root string, walEnabled, snapshotEnabled bool) *testServer {
	t.Helper()
	t.Logf("opening persistence: wal_enabled=%t snapshot_enabled=%t root=%s", walEnabled, snapshotEnabled, root)

	wal, err := storage.OpenWAL(storage.Config{
		Enabled: walEnabled,
		Path:    filepath.Join(root, "wal.log"),
	})
	if err != nil {
		t.Fatalf("open wal: %v", err)
	}

	snapshot, err := storage.OpenSnapshot(storage.Config{
		Enabled: snapshotEnabled,
		Path:    filepath.Join(root, "snapshot.json"),
	})
	if err != nil {
		_ = wal.Close()
		t.Fatalf("open snapshot: %v", err)
	}

	store, err := kv.NewStoreFromDisk(wal, snapshot)
	if err != nil {
		_ = wal.Close()
		t.Fatalf("restore store: %v", err)
	}

	server := httptest.NewServer(apiPkg.NewServer(store).Handler())
	t.Logf("started test HTTP server at %s", server.URL)

	return &testServer{
		server: server,
		wal:    wal,
	}
}

func putJSON(t *testing.T, url string, payload any, wantStatus int) responseBody {
	t.Helper()

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return doRequest(t, req, wantStatus)
}

func getJSON(t *testing.T, url string, wantStatus int) responseBody {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	return doRequest(t, req, wantStatus)
}

func deleteRequest(t *testing.T, url string, wantStatus int) {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	_ = doRequest(t, req, wantStatus)
}

func doRequest(t *testing.T, req *http.Request, wantStatus int) responseBody {
	t.Helper()
	t.Logf("%s %s (expecting %d)", req.Method, req.URL.String(), wantStatus)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", req.Method, req.URL.String(), err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if resp.StatusCode != wantStatus {
		t.Fatalf("unexpected status for %s %s: got %d want %d body=%s", req.Method, req.URL.String(), resp.StatusCode, wantStatus, string(payload))
	}

	if len(payload) == 0 {
		t.Logf("received status %d with empty body", resp.StatusCode)
		return nil
	}

	t.Logf("received status %d body=%s", resp.StatusCode, string(payload))

	var body responseBody
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return body
}

func assertString(t *testing.T, body responseBody, key, want string) {
	t.Helper()

	got, ok := body[key].(string)
	if !ok {
		t.Fatalf("expected %q to be a string, got %#v", key, body[key])
	}
	if got != want {
		t.Fatalf("expected %q=%q, got %q", key, want, got)
	}

	t.Logf("verified %s=%s", key, want)
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected file %s to be non-empty", path)
	}

	t.Logf("verified persisted file exists: %s (%d bytes)", path, info.Size())
}
