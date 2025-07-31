package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"strconv"
)

func RenderRangeAmountArrayChart(rangeAmountArray []*RangeAmount, currentTick, tickSpacing int32, height uint64, token0Symbol, token1Symbol string) (string, error) {
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
        <b>高度:</b> {{.Height}} &nbsp; <b>交易对:</b> {{.Token0Symbol}}/{{.Token1Symbol}} &nbsp; <b>当前 Tick:</b> {{.CurrentTick}} &nbsp; <b>TickSpacing:</b> {{.TickSpacing}} &nbsp; <b>当前 Tick 价格:</b> {{.CurrentPrice}}
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

		function getBarColors(arr, currentTick) {
			return arr.map(rangeAmount => {
				const tickLower = rangeAmount.TickLower;
				const tickUpper = rangeAmount.TickUpper;
				return (currentTick >= tickLower && currentTick < tickUpper)
					? 'rgba(255, 99, 132, 0.8)'
					: 'rgba(54, 162, 235, 0.5)';
			});
		}

        function makeChart(canvasId, arr, currentTick) {
            const {ticks, prices} = getTicksAndPrices(arr);
            const barColors = getBarColors(arr, currentTick);
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
		"Height":           height,
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
