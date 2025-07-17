package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

type APIServer interface {
	Start()
}

type apiServer struct {
	db DBWrap
}

func (a *apiServer) Start() {
	go func() {
		http.HandleFunc("/tickstate", func(w http.ResponseWriter, r *http.Request) {
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
			state, err := a.GetTickState(address, int32(tick))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("get tick state error: %v", err)))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(state)
		})

		http.HandleFunc("/tickstates", func(w http.ResponseWriter, r *http.Request) {
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
			states, err := a.GetTickStates(address, int32(tickLower), int32(tickUpper))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("get tick states error: %v", err)))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(states)
		})

		http.HandleFunc("/ticks/all", func(w http.ResponseWriter, r *http.Request) {
			states, err := a.GetAll()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("get all tick states error: %v", err)))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(states)
		})

		log.Println("API server started at :8080")
		err := http.ListenAndServe(":29999", nil) // TODO config
		if err != nil {
			panic(err)
		}
	}()
}

func (a *apiServer) GetTickState(address common.Address, tick int32) (*TickState, error) {
	key := GetTickStateKey(address, tick)
	tickState, err := a.db.GetTickState(key)
	if err != nil {
		return nil, err
	}
	return tickState, nil
}

func (a *apiServer) GetTickStates(address common.Address, tickLower, tickUpper int32) ([]*TickState, error) {
	keyLower := GetTickStateKey(address, tickLower)
	keyUpper := GetTickStateKey(address, tickUpper)
	tickStates, err := a.db.GetTickStates(keyLower, keyUpper)
	if err != nil {
		return nil, err
	}
	return tickStates, nil
}

var (
	minTick = int32(-8388608) // int24
	maxTick = int32(8388607)  // int24
	minAddr = common.Address{}
	maxAddr = common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff") // 20 bytes of 0xff
)

func (a *apiServer) GetAll() ([]*TickState, error) {
	keyLower := GetTickStateKey(minAddr, minTick)
	keyUpper := GetTickStateKey(maxAddr, maxTick)
	return a.db.GetTickStates(keyLower, keyUpper)
}

func NewAPIServer(db DBWrap) APIServer {
	return &apiServer{
		db: db,
	}
}
