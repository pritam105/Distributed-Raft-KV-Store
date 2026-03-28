package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"distributed-raft-kv-store/kv"
	"distributed-raft-kv-store/storage"
)

func TestServerKeyLifecycle(t *testing.T) {
	server := NewServer(kv.NewStore(storage.NewNoopWAL(), storage.NewNoopSnapshot()))

	putReq := httptest.NewRequest(http.MethodPut, "/v1/keys/user1", bytes.NewBufferString(`{"value":"alice"}`))
	putRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(putRes, putReq)
	if putRes.Code != http.StatusOK {
		t.Fatalf("expected PUT 200, got %d", putRes.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/keys/user1", nil)
	getRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected GET 200, got %d", getRes.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/v1/keys/user1", nil)
	deleteRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected DELETE 204, got %d", deleteRes.Code)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/v1/keys/user1", nil)
	missingRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(missingRes, missingReq)
	if missingRes.Code != http.StatusNotFound {
		t.Fatalf("expected GET after delete 404, got %d", missingRes.Code)
	}
}

func TestServerRejectsNestedKeyPath(t *testing.T) {
	server := NewServer(kv.NewStore(storage.NewNoopWAL(), storage.NewNoopSnapshot()))

	req := httptest.NewRequest(http.MethodGet, "/v1/keys/a/b", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}
