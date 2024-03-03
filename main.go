package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Config struct {
	some    string
	timeout int
}

type ConfigMutex struct {
	cfg Config
	mu  sync.Mutex
}

var (
	cfgPtr            atomic.Pointer[Config]
	cfgM              *ConfigMutex
	atomicCalls       atomic.Uint64
	mutCalls          atomic.Uint64
	averageAtomicTime atomic.Uint64
	averageMutTime    atomic.Uint64
)

var charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func cleanup() {
	fmt.Println("atomicCalls:", atomicCalls.Load())
	fmt.Println("copyCalls:", mutCalls.Load())
	fmt.Println("sum atomic time:", time.Duration(averageAtomicTime.Load()))
	fmt.Println("sum mutex time:", time.Duration(averageMutTime.Load()))

	fmt.Println("average atomic time:", time.Duration(averageAtomicTime.Load()/atomicCalls.Load()))
	fmt.Println("average mutex time:", time.Duration(averageMutTime.Load()/mutCalls.Load()))
}

func randomValue() string {
	b := make([]byte, 10)
	for i := range b {
		b[i] = charSet[rand.Intn(len(charSet))]
	}
	return string(b)
}

func update() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
		go update()
	}()
	for {
		time.Sleep(5 * time.Second)
		cfg := cfgPtr.Load()
		r := randomValue()
		cfg.some = r
		cfgPtr.Store(cfg)
		cfgM.mu.Lock()
		cfgM.cfg.some = r
		cfgM.mu.Unlock()
	}
}

func wrapMutex(cfg *Config) *ConfigMutex {
	return &ConfigMutex{cfg: *cfg}
}

func main() {
	mux := http.NewServeMux()

	cfg := Config{
		some:    "value",
		timeout: 10,
	}

	cfgM = wrapMutex(&cfg)

	cfgPtr.Store(&cfg)

	mux.HandleFunc("/atomic", atomicConfig(&cfg))
	mux.HandleFunc("/mutex", mutexConfig())
	// mux.HandleFunc("/channel", channelConfig())
	// mux.HandleFunc("/once", onceConfig())
	// mux.HandleFunc("/sync", syncConfig())

	go update()

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

func mutexConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		n := time.Now()
		cfgM.mu.Lock()
		defer cfgM.mu.Unlock()
		w.Write([]byte(cfgM.cfg.some))
		fmt.Println("mutex:", time.Since(n))
		mutCalls.Add(1)
		averageMutTime.Add(uint64(time.Since(n).Nanoseconds()))
	}
}
