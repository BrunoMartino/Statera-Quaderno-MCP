package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"woocommerce-store-mcp/internal/config"
	"woocommerce-store-mcp/internal/guard"
	mcpserver "woocommerce-store-mcp/internal/mcp"
)

func main() {
	httpAddr := flag.String("http", "", "listen address for Streamable HTTP (empty = stdio)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg, err := config.NewEnvConfigFactory(os.Getenv).GetConfig()
	if err != nil {
		logger.Error("boot", "err", err)
		os.Exit(1)
	}

	policy := guard.NewAccessPolicy(cfg.AllowedStatuses)
	caller := guard.NewStoreCaller(cfg.BaseURL, cfg.Auth, policy)
	caller.Logger = logger
	caller.StoreID = cfg.StoreID
	caller.Environment = cfg.Environment
	rt := mcpserver.New(cfg, caller, logger)

	if *httpAddr != "" {
		if err := http.ListenAndServe(*httpAddr, rt.HTTPHandler()); err != nil {
			logger.Error("http", "err", err)
			os.Exit(1)
		}
		return
	}

	if err := rt.Server().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Error("stdio", "err", err)
		os.Exit(1)
	}
}
