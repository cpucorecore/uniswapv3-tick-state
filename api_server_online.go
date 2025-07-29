package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"time"
)

const (
	MinTick = -887272
	MaxTick = 887272
)

type APIServer interface {
	Start()
}

type apiServer struct {
	cc    *ContractCaller
	cache Cache
	db    Repo
}

type ParseError struct {
	Code    int
	Message string
}

func parseParams(r *http.Request, requiredParams []string) (map[string]string, *ParseError) {
	params := make(map[string]string)

	for _, param := range requiredParams {
		value := r.URL.Query().Get(param)
		if value == "" {
			return nil, &ParseError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("missing parameter: %s", param),
			}
		}
		params[param] = value
	}

	return params, nil
}

func convertParams(params map[string]string) (common.Address, int32, *ParseError) {
	addressStr, ok := params["address"]
	if !ok {
		return common.Address{}, 0, &ParseError{
			Code:    http.StatusBadRequest,
			Message: "missing address parameter",
		}
	}
	address := common.HexToAddress(addressStr)

	tickOffsetStr, ok := params["tick_offset"]
	if !ok {
		return common.Address{}, 0, &ParseError{
			Code:    http.StatusBadRequest,
			Message: "missing tick_offset parameter",
		}
	}
	tickOffset, err := strconv.ParseInt(tickOffsetStr, 10, 32)
	if err != nil {
		return common.Address{}, 0, &ParseError{
			Code:    http.StatusBadRequest,
			Message: "invalid tick_offset format",
		}
	}

	return address, int32(tickOffset), nil
}

func (a *apiServer) parsePoolStateParams(r *http.Request) (common.Address, int32, *ParseError) {
	params, parseErr := parseParams(r, []string{"address", "tick_offset"})
	if parseErr != nil {
		return common.Address{}, 0, parseErr
	}

	return convertParams(params)
}

func (a *apiServer) HandlerPoolState(w http.ResponseWriter, r *http.Request) {
	address, tickOffset, parseErr := a.parsePoolStateParams(r)
	if parseErr != nil {
		w.WriteHeader(parseErr.Code)
		w.Write([]byte(parseErr.Message))
		return
	}

	pair, ok := a.cache.GetPair(address)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("no pool info"))
		return
	}

	if pair.Filtered {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("pool filtered"))
		return
	}

	poolState, err := GetPoolStateFromDBOrContractCaller(a.db, a.cc, address)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get tick states error: %v", err)))
		return
	}

	token0, token1 := pair.Token0Core, pair.Token1Core
	if pair.TokensReversed {
		token0, token1 = pair.Token1Core, pair.Token0Core
	}

	currentTick := int32(poolState.GlobalState.Tick.Int64())
	tickSpacing := int32(poolState.GlobalState.TickSpacing.Int64())
	centerTick := (currentTick / tickSpacing) * tickSpacing
	tickLower := centerTick - tickOffset*tickSpacing
	tickUpper := centerTick + (tickOffset+1)*tickSpacing

	now := time.Now()
	amount, summary := CalcAmount(poolState.TickStates, tickSpacing, tickLower, tickUpper, int(token0.Decimals), int(token1.Decimals))
	Log.Info("CalcAmount duration", zap.Any("ms", time.Since(now).Milliseconds()))

	now = time.Now()
	htmlStr, err := RenderTickAmountCharts(amount, summary, currentTick, tickSpacing, token0.Symbol, token1.Symbol)
	Log.Info("RenderTickAmountCharts duration", zap.Any("ms", time.Since(now).Milliseconds()))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("render error"))
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlStr))
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

func NewAPIServer(url string, cache Cache, db Repo) APIServer {
	cc := NewContractCaller(url)
	return &apiServer{
		cc:    cc,
		cache: cache,
		db:    db,
	}
}

func RenderTickAmountCharts(amount, summary []TickAmount, currentTick, tickSpacing int32, token0Symbol, token1Symbol string) (string, error) {
	amountJSON, err := json.Marshal(amount)
	if err != nil {
		return "", err
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}

	currentPrice := fmt.Sprintf("%g", float64Pow(1.0001, float64(currentTick), 5))

	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Tick Amount Chart</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <div style="margin-bottom: 16px;">
        <b>交易对:</b> {{.Token0Symbol}}/{{.Token1Symbol}} &nbsp; <b>当前 Tick:</b> {{.CurrentTick}} &nbsp; <b>TickSpacing:</b> {{.TickSpacing}} &nbsp; <b>当前 Tick 价格:</b> {{.CurrentPrice}}
    </div>
    <h2>TickSpace 明细</h2>
    <canvas id="amountChart"></canvas>
    <h2>原始Tick区间合计</h2>
    <canvas id="summaryChart"></canvas>
    <script>
        const amount = {{.Amount}};
        const summary = {{.Summary}};
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

        makeChart('amountChart', amount, currentTick);
        makeChart('summaryChart', summary, currentTick);
    </script>
</body>
</html>
`
	t := template.Must(template.New("chart").Delims("{{", "}}").Parse(html))
	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]interface{}{
		"Amount":       template.JS(amountJSON),
		"Summary":      template.JS(summaryJSON),
		"CurrentTick":  currentTick,
		"TickSpacing":  tickSpacing,
		"CurrentPrice": currentPrice,
		"Token0Symbol": token0Symbol,
		"Token1Symbol": token1Symbol,
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
