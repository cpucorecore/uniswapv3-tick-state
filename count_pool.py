import csv

input_file = 'group_pools.csv'

all_pools = set()
all_tokens = set()

with open(input_file, newline='') as csvfile:
    reader = csv.reader(csvfile)
    header = next(reader)  # 跳过表头
    for row in reader:
        if not row or len(row) < 3:
            continue
        token0 = row[0]
        token1 = row[1]
        pools = row[2].split(';')
        all_tokens.add(token0)
        all_tokens.add(token1)
        for pool in pools:
            all_pools.add(pool)

print(f'不同的pool地址数量: {len(all_pools)}')
print(f'不同的token地址数量: {len(all_tokens)}')

# 导出所有pool和token地址到csv
with open('all_pools.csv', 'w', newline='') as f:
    writer = csv.writer(f)
    writer.writerow(['pool_address'])
    for pool in sorted(all_pools):
        writer.writerow([pool])

with open('all_tokens.csv', 'w', newline='') as f:
    writer = csv.writer(f)
    writer.writerow(['token_address'])
    for token in sorted(all_tokens):
        writer.writerow([token])

# 生成redis key列表
with open('all_redis_keys.txt', 'w') as f:
    for pool in sorted(all_pools):
        f.write(f'npr:{pool}\n')
    for token in sorted(all_tokens):
        f.write(f'nt:{token}\n')

print('已导出所有pool和token地址，以及redis key列表。')