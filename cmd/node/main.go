// Copyright (c) 2026 WabiSaby
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wabisaby/wabisaby-node/internal/container"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.NopLogger,
		container.NodeModule,
	)

	if err := app.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "[node] startup error:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Start(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "[node] start error:", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := app.Stop(shutdownCtx); err != nil {
		fmt.Fprintln(os.Stderr, "[node] shutdown error:", err)
		os.Exit(1)
	}
}
