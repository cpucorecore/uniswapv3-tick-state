import rocksdb

db = rocksdb.DB("/path/to/your/db", rocksdb.Options(create_if_missing=False))
it = db.iterkeys()
it.seek_to_first()
for key in it:
    value = db.get(key)
    print(key, value)