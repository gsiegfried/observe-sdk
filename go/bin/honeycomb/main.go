package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/dylibso/observe-sdk/go/adapter/honeycomb"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func main() {
	ctx := context.Background()

	// we only need to create and start once per instance of our host app
	conf := &honeycomb.HoneycombConfig{
		ApiKey:             os.Getenv("HONEYCOMB_API_KEY"),
		Dataset:            "golang",
		EmitTracesInterval: 1000,
		TraceBatchMax:      100,
		Host:               "https://api.honeycomb.io",
	}
	adapter := honeycomb.NewHoneycombAdapter(conf)
	defer adapter.Stop(true)
	adapter.Start(ctx)

	// Load WASM from disk
	wasm, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Panicln(err)
	}

	cfg := wazero.NewRuntimeConfig().WithCustomSections(true)
	rt := wazero.NewRuntimeWithConfig(ctx, cfg)
	traceCtx, err := adapter.NewTraceCtx(ctx, rt, wasm, nil)
	if err != nil {
		log.Panicln(err)
	}
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	config := wazero.NewModuleConfig().
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithArgs(os.Args[1:]...).
		WithStartFunctions("_start")
	m, err := rt.InstantiateWithConfig(ctx, wasm, config)
	if err != nil {
		log.Panicln(err)
	}
	defer m.Close(ctx)

	// normally this metadata would be in your web-server framework
	// or derived when you need them

	meta := map[string]string{
		"http.url":         "https://example.com/my-endpoint",
		"http.status_code": "200",
		"http.client_ip":   "66.210.227.34",
	}
	traceCtx.Metadata(meta)

	traceCtx.Finish()
	time.Sleep(2 * time.Second)
}
