import csv
from collections import defaultdict

input_file = 'pools.csv'
output_file = 'group_pools.csv'

groups = defaultdict(list)

with open(input_file, newline='') as csvfile:
    reader = csv.reader(csvfile)
    for row in reader:
        if not row or len(row) < 4:
            continue
        pool_address = row[1]
        token0 = row[2]
        token1 = row[3]
        key = (token0, token1)
        groups[key].append(pool_address)

with open(output_file, 'w', newline='') as csvfile:
    writer = csv.writer(csvfile)
    writer.writerow(['token0_address', 'token1_address', 'pool_addresses'])
    for (token0, token1), pool_list in groups.items():
        if len(pool_list) > 1:  # 只保留可套利的分组
            writer.writerow([token0, token1, ';'.join(pool_list)])

print(f'分组完成，结果已保存到 {output_file}（已过滤掉只含一个池子的分组）')