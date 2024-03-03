package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func main() {
	ctx := context.Background()

	go callAtomic(ctx)
	go callMut(ctx)

	// timeout after 5 seconds
	<-time.After(5 * time.Second)
}

func callAtomic(ctx context.Context) {
	defer rec(ctx, callAtomic)
	client := &http.Client{}
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/atomic", nil)
	for {
		client.Do(req)
	}
}

func callMut(ctx context.Context) {
	defer rec(ctx, callMut)
	client := &http.Client{}
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/mutex", nil)
	for {
		client.Do(req)
	}
}

func rec(ctx context.Context, fn func(context.Context)) {
	if r := recover(); r != nil {
		fmt.Println("recovered")
		fn(ctx)
	}
}
