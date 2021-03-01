# redis-de-go
redis serverのgolangによる実装

## 使い方
build
```shell
go build -o redis-de-go main.go
```

help
```shell
./redis-de-go -h
```

run
```
./redis-de-go
```

## 実装コマンド
- ping
- set
- get
- del
- exists
- incrby/decrby
- incr/decr
- rename
- time
- append
- dbsize
- touch
- mget
- mset
