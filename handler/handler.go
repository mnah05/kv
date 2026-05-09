package handler

import (
	"encoding/json"
	"net/http"
)

type Store interface {
	Get(key string) (string, bool)
	Put(key, val string) error
	Delete(key string) error
}

func HandlePut(s Store) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			Key string `json:"key"`
			Val string `json:"val"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(rw, "invalid json", http.StatusBadRequest)
			return
		}
		if body.Key == "" {
			http.Error(rw, "missing key", http.StatusBadRequest)
			return
		}

		if err := s.Put(body.Key, body.Val); err != nil {
			http.Error(rw, "write failed", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.Write([]byte("ok"))
	}
}

func HandleGet(s Store) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(rw, "missing key", http.StatusBadRequest)
			return
		}

		val, ok := s.Get(key)
		if !ok {
			http.NotFound(rw, r)
			return
		}

		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.Write([]byte(val))
	}
}

func HandleDelete(s Store) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(rw, "missing key", http.StatusBadRequest)
			return
		}

		if err := s.Delete(key); err != nil {
			http.Error(rw, "delete failed", http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusNoContent)
	}
}
