package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type APIServerOnline interface {
	Start()
}

type apiServerOnline struct {
	cc    *ContractCaller
	cache Cache
	db    Repo
}

const (
	MinTick = -887272
	MaxTick = 887272
)

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

	tickLower, err1 := strconv.ParseInt(tickLowerStr, 10, 32)
	tickUpper, err2 := strconv.ParseInt(tickUpperStr, 10, 32)
	if err1 != nil || err2 != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid tickLower or tickUpper"))
		return
	}
	Log.Info("req", zap.Int64("tickLower", tickLower), zap.Int64("tickUpper", tickUpper))

	now := time.Now()
	ticks, err := a.cc.CallGetAllTicks(address)
	Log.Info("CallGetAllTicks duration", zap.Any("ms", time.Since(now).Milliseconds()))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get tick states error: %v", err)))
		return
	}
	bytes, _ := json.Marshal(ticks)
	Log.Info(fmt.Sprintf("get tick states: %s", string(bytes)))

	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(ticks)

	token0, token1 := pair.Token0Core, pair.Token1Core
	if pair.TokensReversed {
		token0, token1 = pair.Token1Core, pair.Token0Core
	}
	if tickLower == 0 && tickUpper == 0 {
		tickLower = MinTick
		tickUpper = MaxTick
	}

	now = time.Now()
	amount, summary := CalcAmount(ticks.State, ticks.Ticks, int32(tickLower), int32(tickUpper), int(token0.Decimals), int(token1.Decimals))
	Log.Info("CalcAmount duration", zap.Any("ms", time.Since(now).Milliseconds()))
	//json.NewEncoder(w).Encode(amount)
	//json.NewEncoder(w).Encode(summary)

	now = time.Now()
	htmlStr, err := RenderTickAmountCharts(amount, summary, int32(ticks.State.Tick.Int64()), int32(ticks.State.TickSpacing.Int64()), token0.Symbol, token1.Symbol)
	Log.Info("RenderTickAmountCharts duration", zap.Any("ms", time.Since(now).Milliseconds()))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("render error"))
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlStr))
}

func (a *apiServerOnline) HandlerTicks2(w http.ResponseWriter, r *http.Request) {
	addressStr := r.URL.Query().Get("address")
	tickOffsetStr := r.URL.Query().Get("TickOffset")
	if addressStr == "" || tickOffsetStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing address or TickOffset"))
		return
	}

	address := common.HexToAddress(addressStr)
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

	tickOffset, err := strconv.ParseInt(tickOffsetStr, 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid TickCount"))
		return
	}

	now := time.Now()
	ticks, err := a.cc.CallGetAllTicks(address)
	Log.Info("CallGetAllTicks duration", zap.Any("ms", time.Since(now).Milliseconds()))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get tick states error: %v", err)))
		return
	}

	token0, token1 := pair.Token0Core, pair.Token1Core
	if pair.TokensReversed {
		token0, token1 = pair.Token1Core, pair.Token0Core
	}

	currentTick := int32(ticks.State.Tick.Int64())
	tickSpacing := int32(ticks.State.TickSpacing.Int64())
	// 计算窗口
	centerTick := (currentTick / tickSpacing) * tickSpacing
	tickLower := centerTick - int32(tickOffset)*tickSpacing
	tickUpper := centerTick + (int32(tickOffset)+1)*tickSpacing

	now = time.Now()
	amount, summary := CalcAmount(ticks.State, ticks.Ticks, tickLower, tickUpper, int(token0.Decimals), int(token1.Decimals))
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

func (a *apiServerOnline) HandlerTicks3(w http.ResponseWriter, r *http.Request) {
	// 查询address是否初始化
	// 如果没有初始化,调用CallGetAllTicks获取当前pool的所有tick信息并保存到db(包含tickSpacing),得到所有tick信息:ticks;同时event_reactor开始处理该address的事件,根据事件(Mint/Burn)更新tick状态,根据事件(Swap)更新currentTick(问题,获取到ticks信息的高度是H,此时主流程的高度已经处理到H+2,就会丢失2个区块的状态)
	// 如果初始化,直接调用db的接口获取当前所有tick信息:ticks
	// 到这里为止无论是否初始化最终都得到了池子的ticks
	// 根据ticks,currentTick,TickSpacing计算深度并返回结果

	// 抽象出一个函数:ParseRequest得到address,tickOffset
	// 查询db:
	/*
		db需要存储的数据:
		当前Tick(实时更新),TickSpacing(一次写入不可变)
		各Tick具体状态{LiquidityNet}(实时更新)
	*/
	addressStr := r.URL.Query().Get("address")
	tickOffsetStr := r.URL.Query().Get("TickOffset")
	if addressStr == "" || tickOffsetStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing address or TickOffset"))
		return
	}

	address := common.HexToAddress(addressStr)
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

	tickOffset, err := strconv.ParseInt(tickOffsetStr, 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid TickCount"))
		return
	}

	now := time.Now()

	ticks, err := GetAndGet(a.db, a.cc, address)
	bs, _ := json.Marshal(ticks)
	Log.Info("GetAndGetTicks duration", zap.Any("ms", time.Since(now).Milliseconds()), zap.String("addr", addressStr), zap.String("ts", string(bs)))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get tick states error: %v", err)))
		return
	}

	token0, token1 := pair.Token0Core, pair.Token1Core
	if pair.TokensReversed {
		token0, token1 = pair.Token1Core, pair.Token0Core
	}

	currentTick := int32(ticks.State.Tick.Int64())
	tickSpacing := int32(ticks.State.TickSpacing.Int64())
	// 计算窗口
	centerTick := (currentTick / tickSpacing) * tickSpacing
	tickLower := centerTick - int32(tickOffset)*tickSpacing
	tickUpper := centerTick + (int32(tickOffset)+1)*tickSpacing

	now = time.Now()
	amount, summary := CalcAmount(ticks.State, ticks.Ticks, tickLower, tickUpper, int(token0.Decimals), int(token1.Decimals))
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

func (a *apiServerOnline) Start() {
	go func() {
		http.HandleFunc("/online/ticks", a.HandlerTicks)
		http.HandleFunc("/online/ticks2", a.HandlerTicks2)
		http.HandleFunc("/online/ticks3", a.HandlerTicks3)
		err := http.ListenAndServe(":39999", nil)
		if err != nil {
			panic(err)
		}
	}()
}

func NewAPIServerOnline(url string, cache Cache, db Repo) APIServer {
	cc := NewContractCaller(url)
	return &apiServerOnline{
		cc:    cc,
		cache: cache,
		db:    db,
	}
}

// RenderTickAmountCharts 生成包含两个图表的HTML
func RenderTickAmountCharts(amount, summary []TickAmount, currentTick, tickSpacing int32, token0Symbol, token1Symbol string) (string, error) {
	amountJSON, err := json.Marshal(amount)
	if err != nil {
		return "", err
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}

	// 计算当前tick对应的价格，保留5位有效数字
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

// float64Pow returns base^exp, 保留n位有效数字
func float64Pow(base, exp float64, precision int) float64 {
	v := math.Pow(base, exp)
	// 保留n位有效数字
	format := "%." + fmt.Sprintf("%d", precision) + "g"
	res, _ := strconv.ParseFloat(fmt.Sprintf(format, v), 64)
	return res
}
