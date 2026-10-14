package kv

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"strconv"
)

func opFromRequest(r *http.Request, typ, key, value string) Op {
	cid, err1 := strconv.ParseInt(r.Header.Get("X-Client-Id"), 10, 64)
	seq, err2 := strconv.ParseInt(r.Header.Get("X-Seq"), 10, 64)
	if err1 != nil || err2 != nil {
		cid = rand.Int63()
		seq = 1
	}
	return Op{Type: typ, Key: key, Value: value, ClientID: cid, Seq: seq}
}

func writeErr(w http.ResponseWriter, res Result) bool {
	switch res.Err {
	case "":
		return false
	case ErrTimeout:
		http.Error(w, res.Err, http.StatusGatewayTimeout)
	default:
		http.Error(w, res.Err, http.StatusServiceUnavailable)
	}
	return true
}

func (s *Server) mutation(typ string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res := s.Submit(opFromRequest(r, typ, r.PathValue("key"), string(body)))
		if writeErr(w, res) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /kv/{key}", func(w http.ResponseWriter, r *http.Request) {
		res := s.Submit(opFromRequest(r, OpGet, r.PathValue("key"), ""))
		if writeErr(w, res) {
			return
		}
		if !res.Ok {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, res.Value)
	})
	mux.HandleFunc("PUT /kv/{key}", s.mutation(OpPut))
	mux.HandleFunc("POST /kv/{key}", s.mutation(OpAppend))
	mux.HandleFunc("DELETE /kv/{key}", s.mutation(OpDelete))
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		term, leader := s.rf.GetState()
		json.NewEncoder(w).Encode(map[string]any{"term": term, "leader": leader})
	})
	return mux
}