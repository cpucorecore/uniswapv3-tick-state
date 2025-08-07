import csv
import requests
import json
from itertools import combinations
import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from tqdm import tqdm
import signal
import sys

API_URL = 'http://localhost:29292/arbitrage_check'
INPUT_FILE = 'group_pools_filtered.csv'
JSON_REPORT = 'arbitrage_report.json'
HTML_REPORT = 'arbitrage_report.html'

parser = argparse.ArgumentParser(description='批量套利分析')
parser.add_argument('--max-groups', '-n', type=int, default=None, help='最多处理多少个group（默认全量）')
parser.add_argument('--threads', '-t', type=int, default=8, help='并发线程数')
args = parser.parse_args()

results = []
all_tasks = []
group_count = 0

# 1. 读取分组信息，先生成所有两两组合
with open(INPUT_FILE, newline='') as csvfile:
    reader = csv.reader(csvfile)
    header = next(reader)
    for row in reader:
        if not row or len(row) < 3:
            continue
        token0, token1, pools_str = row
        pools = [p for p in pools_str.split(';') if p]
        for pool1, pool2 in combinations(pools, 2):
            all_tasks.append((token0, token1, pool1, pool2))
        group_count += 1
        if args.max_groups is not None and group_count >= args.max_groups:
            print(f'已处理 {group_count} 个group，提前结束')
            break

total = len(all_tasks)
print(f'总任务数: {total}')

stop_flag = False

def signal_handler(sig, frame):
    global stop_flag
    print('\n检测到中断信号，正在保存已完成的结果...')
    stop_flag = True
signal.signal(signal.SIGINT, signal_handler)

def process_pair(token0, token1, pool1, pool2):
    params = {'pool1': pool1, 'pool2': pool2}
    try:
        resp = requests.get(API_URL, params=params, timeout=10)
        if resp.status_code == 200:
            data = resp.json()
            profit = float(data.get('max_profit') or data.get('profit') or 0)
            return {
                'token0': token0,
                'token1': token1,
                'pool1': pool1,
                'pool2': pool2,
                'profit': profit,
                'api_result': data
            }
        else:
            print(f'API error for {pool1}, {pool2}: {resp.text}')
    except Exception as e:
        print(f'Exception for {pool1}, {pool2}: {e}')
    return None

def save_reports(results):
    # 按绝对利润排序
    results.sort(key=lambda x: abs(x['profit']), reverse=True)
    # 保存JSON报告
    with open(JSON_REPORT, 'w') as f:
        json.dump(results, f, indent=2, ensure_ascii=False)
    print(f'JSON报告已保存到 {JSON_REPORT}')
    # 生成HTML报告
    with open(HTML_REPORT, 'w') as f:
        f.write('<html><head><meta charset="utf-8"><title>Arbitrage Report</title></head><body>')
        f.write('<h1>Uniswap V3 跨池套利分析报告</h1>')
        f.write('<table border="1" cellpadding="4" cellspacing="0">')
        f.write('<tr><th>Token0</th><th>Token1</th><th>Pool1</th><th>Pool2</th><th>利润(USD)</th><th>详情</th></tr>')
        for r in results:
            f.write(
                f'<tr><td>{r["token0"]}</td><td>{r["token1"]}</td><td>{r["pool1"]}</td><td>{r["pool2"]}</td>'
                f'<td>{r["profit"]:.6f}</td>'
                f'<td><pre style="white-space:pre-wrap">{json.dumps(r["api_result"], ensure_ascii=False, indent=2)}</pre></td></tr>'
            )
        f.write('</table></body></html>')
    print(f'HTML报告已保存到 {HTML_REPORT}')

# 2. 多线程+进度条处理
try:
    with ThreadPoolExecutor(max_workers=args.threads) as executor:
        futures = [executor.submit(process_pair, token0, token1, pool1, pool2) for token0, token1, pool1, pool2 in all_tasks]
        for f in tqdm(as_completed(futures), total=total, desc='套利分析'):
            if stop_flag:
                break
            r = f.result()
            if r:
                results.append(r)
except KeyboardInterrupt:
    print('\n用户中断，正在保存已完成的结果...')
finally:
    save_reports(results)