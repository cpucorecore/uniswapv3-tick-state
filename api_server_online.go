package main

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type APIServerOnline interface {
	Start()
}

type apiServerOnline struct {
	cc *ContractCaller
}

func (a *apiServerOnline) HandlerTicks(w http.ResponseWriter, r *http.Request) {
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
	Log.Info("req", zap.Int64("tickLower", tickLower), zap.Int64("tickUpper", tickUpper))

	ticks, err := a.cc.CallGetAllTicks(address)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get tick states error: %v", err)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticks)
}

func (a *apiServerOnline) Start() {
	go func() {
		http.HandleFunc("/online/ticks", a.HandlerTicks)
		err := http.ListenAndServe(":39999", nil)
		if err != nil {
			panic(err)
		}
	}()
}

func NewAPIServerOnline(url string) APIServer {
	cc := NewContractCaller(url)
	return &apiServerOnline{
		cc: cc,
	}
}
