package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

type Config struct {
	some    string
	timeout int
}

var (
	cfgPtr            atomic.Pointer[Config]
	atomicCalls       atomic.Uint64
	copyCalls         atomic.Uint64
	averageAtomicTime atomic.Uint64
	averageCopyTime   atomic.Uint64
)

func cleanup() {
	fmt.Println("atomicCalls:", atomicCalls.Load())
	fmt.Println("copyCalls:", copyCalls.Load())
	fmt.Println("sum atomic time:", time.Duration(averageAtomicTime.Load()))
	fmt.Println("sunm copy time:", time.Duration(averageCopyTime.Load()))

	fmt.Println("average atomic time:", time.Duration(averageAtomicTime.Load()/atomicCalls.Load()))
	fmt.Println("average copy time:", time.Duration(averageCopyTime.Load()/copyCalls.Load()))
}

func main() {
	mux := http.NewServeMux()

	cfg := Config{
		some:    "value",
		timeout: 10,
	}

	cfgPtr.Store(&cfg)

	mux.HandleFunc("/atomic", atomicConfig(&cfg))
	// mux.HandleFunc("/mutex", mutexConfig())
	// mux.HandleFunc("/channel", channelConfig())
	// mux.HandleFunc("/once", onceConfig())
	// mux.HandleFunc("/sync", syncConfig())
	mux.HandleFunc("/copy", copyConfig(cfg))

	go updateConfig()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()

	http.ListenAndServe(":8080", mux)
}

func atomicConfig(cfg *Config) http.HandlerFunc {
	ptr := atomic.Pointer[Config]{}
	ptr.Store(cfg)

	return func(w http.ResponseWriter, _ *http.Request) {
		n := time.Now()
		cfg := ptr.Load()
		w.Write([]byte(cfg.some))
		fmt.Println("atomic:", time.Since(n))
		atomicCalls.Add(1)
		averageAtomicTime.Add(uint64(time.Since(n).Nanoseconds()))
	}
}

func copyConfig(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		n := time.Now()
		w.Write([]byte(cfg.some))
		fmt.Println("copy", time.Since(n))
		copyCalls.Add(1)
		averageCopyTime.Add(uint64(time.Since(n).Nanoseconds()))
	}
}

func updateConfig() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
		go updateConfig()
	}()
	for {
		time.Sleep(5 * time.Second)
		cfg := cfgPtr.Load()
		cfg.some = "new value"
		cfgPtr.Store(cfg)
	}
}
