package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cdinu/myuplink-naive-client/internal/app"
	"github.com/cdinu/myuplink-naive-client/internal/config"
)

func main() {
	flag.CommandLine.SetOutput(io.Discard)
	configPath := flag.String("config", "config.json", "path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		_ = writeFallbackLog(*configPath, fmt.Sprintf("load config: %v", err))
		os.Exit(1)
	}

	logger, logFile, err := buildLogger(cfg.LogsPath)
	if err != nil {
		_ = writeFallbackLog(*configPath, fmt.Sprintf("setup logger: %v", err))
		os.Exit(1)
	}
	defer logFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := app.Run(ctx, cfg, logger); err != nil {
		logger.Printf("run error: %v", err)
		os.Exit(1)
	}
}

func buildLogger(logsPath string) (*log.Logger, *os.File, error) {
	if err := os.MkdirAll(logsPath, 0o755); err != nil {
		return nil, nil, fmt.Errorf("ensure logs directory: %w", err)
	}

	filename := fmt.Sprintf("nibs-fetch.%s.log", time.Now().UTC().Format("2006-01-02"))
	path := filepath.Join(logsPath, filename)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	logger := log.New(f, "", log.LstdFlags|log.LUTC)
	return logger, f, nil
}

func writeFallbackLog(configPath, message string) error {
	dir := filepath.Dir(configPath)
	if dir == "" || dir == "." {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	filename := fmt.Sprintf("nibs-fetch.%s.log", time.Now().UTC().Format("2006-01-02"))
	path := filepath.Join(dir, filename)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags|log.LUTC)
	logger.Printf("%s", message)
	return nil
}
