package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

const (
	MinTick = -887272
	MaxTick = 887272
)

type APIServer interface {
	Start()
}

type apiServer struct {
	poolStateGetter PoolStateGetter
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
	ParamTypeLiquidity         = "liquidity"
	ParamTypeTokenAmount       = "token_amount"
	ParamTypeTokenAmountDetail = "token_amount_detail"
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
		}
		rangeAmountArray := CalcRangeAmountArray(rangeLiquidityArray, fromTick, toTick, int(poolState.Token0.Decimals), int(poolState.Token1.Decimals))

		if params.Format == "json" {
			w.Header().Set("Content-Type", "application/json")
			jsonData, _ := json.Marshal(rangeAmountArray)
			w.Write(jsonData)
			return
		} else {
			htmlStr, err := RenderRangeAmountArrayChart(rangeAmountArray, int32(poolState.Global.Tick.Int64()), int32(poolState.Global.TickSpacing.Int64()), poolState.Token0.Symbol, poolState.Token1.Symbol)
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

func (a *apiServer) Start() {
	go func() {
		http.HandleFunc("/pool_state", a.HandlerPoolState)
		err := http.ListenAndServe(":29292", nil)
		if err != nil {
			panic(err)
		}
	}()
}

func NewAPIServer(poolStateGetter PoolStateGetter) APIServer {
	return &apiServer{
		poolStateGetter: poolStateGetter,
	}
}

func RenderRangeAmountArrayChart(rangeAmountArray []*RangeAmount, currentTick, tickSpacing int32, token0Symbol, token1Symbol string) (string, error) {
	rangeAmountJSON, err := json.Marshal(rangeAmountArray)
	if err != nil {
		return "", err
	}

	currentPrice := fmt.Sprintf("%g", float64Pow(1.0001, float64(currentTick), 5))

	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Range Amount Chart</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <div style="margin-bottom: 16px;">
        <b>交易对:</b> {{.Token0Symbol}}/{{.Token1Symbol}} &nbsp; <b>当前 Tick:</b> {{.CurrentTick}} &nbsp; <b>TickSpacing:</b> {{.TickSpacing}} &nbsp; <b>当前 Tick 价格:</b> {{.CurrentPrice}}
    </div>
    <h2>Range Amount Chart</h2>
    <canvas id="rangeAmountChart"></canvas>
    <script>
        const rangeAmountArray = {{.RangeAmountArray}};
        const currentTick = {{.CurrentTick}};
		const tickSpacing = {{.TickSpacing}};

        function getTicksAndPrices(arr) {
            const ticks = arr.map(x => x.TickLower);
            const prices = ticks.map(tick => Number(Math.pow(1.0001, tick).toPrecision(5)));
            return {ticks, prices};
        }

		function getBarColors(ticks, tickSpacing, currentTick) {
			return ticks.map(tickLower => {
				const tickUpper = tickLower + tickSpacing;
				return (currentTick >= tickLower && currentTick <= tickUpper)
					? 'rgba(255, 99, 132, 0.8)'
					: 'rgba(54, 162, 235, 0.5)';
			});
		}

        function makeChart(canvasId, arr, currentTick) {
            const {ticks, prices} = getTicksAndPrices(arr);
            const barColors = getBarColors(ticks, tickSpacing, currentTick);
            const data = {
                labels: ticks,
                datasets: [{
                    label: 'amount0',
                    data: arr.map(x => Number(x.Amount0)),
                    backgroundColor: barColors
                }]
            };
            new Chart(document.getElementById(canvasId), {
                type: 'bar',
                data: data,
                options: {
                    scales: {
                        x: {
                            title: { display: true, text: 'Tick' },
                            ticks: {
                                callback: function(value, index) {
                                    return ticks[index];
                                }
                            }
                        },
                        x2: {
                            position: 'top',
                            title: { display: true, text: 'Price' },
                            grid: { drawOnChartArea: false },
                            ticks: {
                                callback: function(value, index) {
                                    return prices[index];
                                }
                            }
                        },
                        y: {
                            title: { display: true, text: 'Amount0' }
                        }
                    }
                }
            });
        }

        makeChart('rangeAmountChart', rangeAmountArray, currentTick);
    </script>
</body>
</html>
`
	t := template.Must(template.New("chart").Delims("{{", "}}").Parse(html))
	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]interface{}{
		"RangeAmountArray": template.JS(rangeAmountJSON),
		"CurrentTick":      currentTick,
		"TickSpacing":      tickSpacing,
		"CurrentPrice":     currentPrice,
		"Token0Symbol":     token0Symbol,
		"Token1Symbol":     token1Symbol,
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func float64Pow(base, exp float64, precision int) float64 {
	v := math.Pow(base, exp)
	format := "%." + fmt.Sprintf("%d", precision) + "g"
	res, _ := strconv.ParseFloat(fmt.Sprintf(format, v), 64)
	return res
}
