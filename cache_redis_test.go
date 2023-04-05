package main

import (
    "context"
    "fmt"
    "github.com/redis/go-redis/v9"
    "testing"
    "time"
)

var ctx = context.Background()

func Test_Redis_GetSet(t *testing.T) {
    rds := redis.NewClient(&redis.Options{
        Addr:     "127.0.0.1:6379",
        Password: "",
        DB:       0,
    })

    err := rds.SetArgs(ctx, "key", []byte("BYTES_ORIGIN_STRING"),
        redis.SetArgs{Mode: "NX", TTL: time.Second * 15}).Err()
    if err != nil {
        panic(err)
    }

    val, err := rds.Get(ctx, "key").Bytes()
    if err != nil {
        panic(err)
    }
    fmt.Println("key", val)
}
