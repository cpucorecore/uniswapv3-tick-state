package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"html/template"
	"net/http"
	"strconv"
)

type APIServerOnline interface {
	Start()
}

type apiServerOnline struct {
	cc    *ContractCaller
	cache Cache
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

	ticks, err := a.cc.CallGetAllTicks(address)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("get tick states error: %v", err)))
		return
	}

	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(ticks)

	token0, token1 := pair.Token0Core, pair.Token1Core
	if pair.TokensReversed {
		token0, token1 = pair.Token1Core, pair.Token0Core
	}
	amount, summary := CalcAmount(ticks.State, ticks.Ticks, int(token0.Decimals), int(token1.Decimals))
	//json.NewEncoder(w).Encode(amount)
	//json.NewEncoder(w).Encode(summary)

	htmlStr, err := RenderTickAmountCharts(amount, summary)
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
		err := http.ListenAndServe(":39999", nil)
		if err != nil {
			panic(err)
		}
	}()
}

func NewAPIServerOnline(url string, cache Cache) APIServer {
	cc := NewContractCaller(url)
	return &apiServerOnline{
		cc:    cc,
		cache: cache,
	}
}

// RenderTickAmountCharts 生成包含两个图表的HTML
func RenderTickAmountCharts(amount, summary []TickAmount) (string, error) {
	amountJSON, err := json.Marshal(amount)
	if err != nil {
		return "", err
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}

	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Tick Amount Chart</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <h2>TickSpace 明细</h2>
    <canvas id="amountChart"></canvas>
    <h2>原始Tick区间合计</h2>
    <canvas id="summaryChart"></canvas>
    <script>
        const amount = {{.Amount}};
        const summary = {{.Summary}};

        function getData(arr) {
            return {
                labels: arr.map(x => x.TickLower),
                datasets: [{
                    label: 'amount0',
                    data: arr.map(x => Number(x.Amount0)),
                    backgroundColor: 'rgba(54, 162, 235, 0.5)'
                }]
            }
        }

        new Chart(document.getElementById('amountChart'), {
            type: 'bar',
            data: getData(amount),
            options: {scales: {x: {title: {display: true, text: 'TickLower'}}, y: {title: {display: true, text: 'Amount0'}}}}
        });

        new Chart(document.getElementById('summaryChart'), {
            type: 'bar',
            data: getData(summary),
            options: {scales: {x: {title: {display: true, text: 'TickLower'}}, y: {title: {display: true, text: 'Amount0'}}}}
        });
    </script>
</body>
</html>
`
	t := template.Must(template.New("chart").Delims("{{", "}}").Parse(html))
	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]interface{}{
		"Amount":  template.JS(amountJSON),
		"Summary": template.JS(summaryJSON),
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
