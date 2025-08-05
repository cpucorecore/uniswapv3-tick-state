import csv

input_file = 'group_pools.csv'
not_found_file = 'not_found_pools.txt'
output_file = 'group_pools_filtered.csv'

# 读取不存在的pool地址
with open(not_found_file) as f:
    not_found_pools = set(line.strip() for line in f if line.strip())

with open(input_file, newline='') as csvfile, open(output_file, 'w', newline='') as outfile:
    reader = csv.reader(csvfile)
    writer = csv.writer(outfile)
    header = next(reader)
    writer.writerow(header)
    filtered_group_count = 0
    for row in reader:
        if not row or len(row) < 3:
            continue
        token0 = row[0]
        token1 = row[1]
        pools = [p for p in row[2].split(';') if p and p not in not_found_pools]
        if len(pools) > 1:
            writer.writerow([token0, token1, ';'.join(pools)])
        else:
            filtered_group_count += 1

print(f'过滤完成，结果已保存到 {output_file}，被过滤掉的分组数量: {filtered_group_count}')