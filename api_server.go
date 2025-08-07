package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

type APIServer interface {
	Start()
}

type apiServer struct {
	poolStateGetter   PoolStateGetter
	arbitrageAnalyzer *ArbitrageAnalyzer
}

func parseParams(r *http.Request, requiredParams []string) (map[string]string, error) {
	params := make(map[string]string)

	for _, param := range requiredParams {
		value := r.URL.Query().Get(param)
		if value == "" {
			return nil, errors.New(param + " is required")
		}
		params[param] = value
	}

	return params, nil
}

type PoolStateParams struct {
	Address    common.Address `json:"address"`
	TickOffset uint64         `json:"tick_offset"`
	Type       string         `json:"type"`
	Format     string         `json:"format"`
}

const (
	ParamAddress    = "address"
	ParamTickOffset = "tick_offset"
	ParamType       = "type"
	ParamFormat     = "format"
)

const (
	ParamTypeLiquidity         = "1" // liquidity
	ParamTypeTokenAmount       = "2" // token_amount
	ParamTypeTokenAmountDetail = "3" // token_amount_detail
)

var (
	ParamList = []string{
		ParamAddress,
		ParamTickOffset,
		ParamType,
		ParamFormat,
	}
)

func FromHttpRequest(r *http.Request) (*PoolStateParams, error) {
	kv, err := parseParams(r, ParamList)
	if err != nil {
		return nil, err
	}

	tickOffset, err := strconv.ParseUint(kv[ParamTickOffset], 10, 32)
	if err != nil {
		return nil, err
	}

	p := &PoolStateParams{
		Address:    common.HexToAddress(kv[ParamAddress]),
		TickOffset: tickOffset,
		Type:       kv[ParamType],
		Format:     kv[ParamFormat],
	}
	p.arrange()
	return p, nil
}

func (p *PoolStateParams) arrange() {
	if p.Type != ParamTypeLiquidity && p.Type != ParamTypeTokenAmount && p.Type != ParamTypeTokenAmountDetail {
		p.Type = ParamTypeLiquidity
	}
	if p.Format != "json" && p.Format != "html" {
		p.Format = "json"
	}
	if p.Type == ParamTypeLiquidity {
		p.Format = "json"
	}
	if p.TickOffset == 0 {
		p.TickOffset = 10
	}
}

func (a *apiServer) HandlerPoolState(w http.ResponseWriter, r *http.Request) {
	params, err := FromHttpRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("wrong request params: %v", err)))
		return
	}

	poolState, err := a.poolStateGetter.GetPoolState(params.Address)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get pool states error: %v", err)))
		return
	}
	Log.Info(fmt.Sprintf("get pool states: %s", poolState))

	switch params.Type {
	case ParamTypeLiquidity:
		w.Header().Add("Content-Type", "application/json")
		w.Write(poolState.Json())
		return

	case ParamTypeTokenAmount, ParamTypeTokenAmountDetail:
		currentTick := int32(poolState.Global.Tick.Int64())
		tickSpacing := int32(poolState.Global.TickSpacing.Int64())
		fromTick, toTick := CalculateTickRange(currentTick, int32(params.TickOffset), tickSpacing)

		rangeLiquidityArray := BuildRangeLiquidityArray(poolState.TickStates)
		rangeLiquidityArray = FilterRangeLiquidityArray(rangeLiquidityArray, fromTick, toTick)
		if params.Type == ParamTypeTokenAmountDetail {
			rangeLiquidityArray = SplitRangeLiquidityArray(rangeLiquidityArray, tickSpacing)
			rangeLiquidityArray = FilterRangeLiquidityArray(rangeLiquidityArray, fromTick, toTick)
		}
		rangeAmountArray := CalcRangeAmountArray(rangeLiquidityArray, int(poolState.Token0.Decimals), int(poolState.Token1.Decimals))

		if params.Format == "json" {
			w.Header().Set("Content-Type", "application/json")
			jsonData, _ := json.Marshal(rangeAmountArray)
			w.Write(jsonData)
			return
		} else {
			htmlStr, err := RenderRangeAmountArrayChart(rangeAmountArray, currentTick, tickSpacing, uint64(poolState.Global.Height.Int64()), poolState.Token0.Symbol, poolState.Token1.Symbol)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("render error"))
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(htmlStr))
			return
		}
	}
}

func (a *apiServer) HandlerArbitrageCheck(w http.ResponseWriter, r *http.Request) {
	pool1Addr := r.URL.Query().Get("pool1")
	pool2Addr := r.URL.Query().Get("pool2")

	if pool1Addr == "" || pool2Addr == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing parameter: pool1 or pool2"))
		return
	}

	pool1 := common.HexToAddress(pool1Addr)
	pool2 := common.HexToAddress(pool2Addr)

	analysis := a.arbitrageAnalyzer.AnalyzeArbitrage(pool1, pool2)
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.Marshal(analysis)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("json marshal error"))
		return
	}
	w.Write(jsonData)
}

func (a *apiServer) Start() {
	go func() {
		http.HandleFunc("/pool_state", a.HandlerPoolState)
		http.HandleFunc("/arbitrage_check", a.HandlerArbitrageCheck)
		err := http.ListenAndServe(":29292", nil)
		if err != nil {
			panic(err)
		}
	}()
}

func NewAPIServer(poolStateGetter PoolStateGetter) APIServer {
	return &apiServer{
		poolStateGetter:   poolStateGetter,
		arbitrageAnalyzer: NewArbitrageAnalyzer(poolStateGetter),
	}
}
