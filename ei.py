import redis

prod_redis = redis.StrictRedis(
    host='localhost',
    port=16379,
    password='',
    decode_responses=True
)

local_redis = redis.StrictRedis(
    host='localhost',
    port=6379,
    decode_responses=True
)

with open('all_redis_keys.txt') as f:
    keys = [line.strip() for line in f if line.strip()]

for key in keys:
    print(key)
    value = prod_redis.get(key)
    if value is not None:
        local_redis.set(key, value)
        print(f'Migrated {key}')
    else:
        print(f'Key not found: {key}')

print('全部string类型key迁移完成！')