package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"distributed-raft-kv-store/kv"
)

type Server struct {
	store *kv.Store
	mux   *http.ServeMux
}

type upsertRequest struct {
	Value string `json:"value"`
}

type keyResponse struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
	Found bool   `json:"found"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewServer(store *kv.Store) *Server {
	server := &Server{
		store: store,
		mux:   http.NewServeMux(),
	}

	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/v1/keys/", s.handleKey)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleKey(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/v1/keys/")
	if key == "" || strings.Contains(key, "/") {
		writeError(w, http.StatusBadRequest, "key must be a single path segment")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getKey(w, key)
	case http.MethodPut:
		s.putKey(w, r, key)
	case http.MethodDelete:
		s.deleteKey(w, key)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) getKey(w http.ResponseWriter, key string) {
	value, found, err := s.store.Get(key)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "key not found")
		return
	}

	writeJSON(w, http.StatusOK, keyResponse{
		Key:   key,
		Value: value,
		Found: true,
	})
}

func (s *Server) putKey(w http.ResponseWriter, r *http.Request, key string) {
	defer r.Body.Close()

	var req upsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request body")
		return
	}

	if err := s.store.Upsert(key, req.Value); err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, keyResponse{
		Key:   key,
		Value: req.Value,
		Found: true,
	})
}

func (s *Server) deleteKey(w http.ResponseWriter, key string) {
	if err := s.store.Delete(key); err != nil {
		writeStoreError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, kv.ErrEmptyKey) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeError(w, http.StatusInternalServerError, err.Error())
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
