package main

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"net/http"
	"strconv"
)

type APIServer interface {
	Start()
}

type apiServer struct {
	db Repo
}

func (a *apiServer) HandlerTick(w http.ResponseWriter, r *http.Request) {
	addressStr := r.URL.Query().Get("address")
	tickStr := r.URL.Query().Get("tick")
	if addressStr == "" || tickStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing address or tick"))
		return
	}

	address := common.HexToAddress(addressStr)
	tick, err := strconv.ParseInt(tickStr, 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid tick"))
		return
	}

	state, err := a.db.GetTickState(address, int32(tick))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get tick state error: %v", err)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func (a *apiServer) HandlerTicks(w http.ResponseWriter, r *http.Request) {
	addressStr := r.URL.Query().Get("address")
	tickLowerStr := r.URL.Query().Get("tickLower")
	tickUpperStr := r.URL.Query().Get("tickUpper")
	if addressStr == "" || tickLowerStr == "" || tickUpperStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing address or tickLower or tickUpper"))
		return
	}

	address := common.HexToAddress(addressStr)
	tickLower, err1 := strconv.ParseInt(tickLowerStr, 10, 32)
	tickUpper, err2 := strconv.ParseInt(tickUpperStr, 10, 32)
	if err1 != nil || err2 != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid tickLower or tickUpper"))
		return
	}

	states, err := a.db.GetTickStates(address, int32(tickLower), int32(tickUpper))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get tick states error: %v", err)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(states)
}

func (a *apiServer) HandlerAll(w http.ResponseWriter, r *http.Request) {
	states, err := a.db.GetAllTicks()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get all tick states error: %v", err)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(states)
}

func (a *apiServer) Start() {
	go func() {
		http.HandleFunc("/tick", a.HandlerTick)
		http.HandleFunc("/ticks", a.HandlerTicks)
		http.HandleFunc("/all", a.HandlerAll)
		err := http.ListenAndServe(":29999", nil)
		if err != nil {
			panic(err)
		}
	}()
}

func NewAPIServer(db Repo) APIServer {
	return &apiServer{
		db: db,
	}
}
