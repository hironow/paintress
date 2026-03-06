package session

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type DevServer struct {
	cmd     string
	url     string
	dir     string
	logPath string
	logger  domain.Logger

	mu      sync.Mutex
	process *exec.Cmd
	running bool
}

func NewDevServer(cmd, url, dir, logPath string, logger domain.Logger) *DevServer {
	if logger == nil {
		logger = &domain.NopLogger{}
	}
	return &DevServer{cmd: cmd, url: url, dir: dir, logPath: logPath, logger: logger}
}

func (ds *DevServer) isRunning() bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.running
}

func (ds *DevServer) setRunning(v bool) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.running = v
}

func (ds *DevServer) Start(ctx context.Context) error {
	ctx, span := platform.Tracer.Start(ctx, "devserver.start",
		trace.WithAttributes(
			attribute.String("cmd", ds.cmd),
			attribute.String("url", ds.url),
		),
	)
	defer span.End()

	if ds.isRunning() {
		span.AddEvent("devserver.ready", trace.WithAttributes(attribute.Bool("external", false)))
		return nil
	}

	// Check if dev server is already running externally
	client := &http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get(ds.url); err == nil {
		resp.Body.Close()
		ds.setRunning(true)
		span.AddEvent("devserver.ready", trace.WithAttributes(attribute.Bool("external", true)))
		ds.logger.OK("%s", fmt.Sprintf(domain.Msg("devserver_already"), ds.url))
		return nil
	}

	ds.logger.Info("%s", fmt.Sprintf(domain.Msg("devserver_start"), ds.cmd, ds.dir))
	logFile, err := os.Create(ds.logPath)
	if err != nil {
		return fmt.Errorf("log file creation failed: %w", err)
	}

	parts := strings.Fields(ds.cmd)
	ds.process = exec.CommandContext(ctx, parts[0], parts[1:]...)
	ds.process.Dir = ds.dir
	ds.process.Stdout = logFile
	ds.process.Stderr = logFile

	if err := ds.process.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("dev server start failed: %w", err)
	}

	go func() {
		defer logFile.Close()
		_ = ds.process.Wait()
		ds.setRunning(false)
	}()

	if err := ds.waitReady(ctx); err != nil {
		ds.Stop()
		return err
	}

	ds.setRunning(true)
	span.AddEvent("devserver.ready", trace.WithAttributes(attribute.Bool("external", false)))
	ds.logger.OK("%s", fmt.Sprintf(domain.Msg("devserver_ready"), ds.url))
	return nil
}

func (ds *DevServer) waitReady(ctx context.Context) error {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	deadline := time.After(60 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("%s", domain.Msg("devserver_timeout"))
		case <-ticker.C:
			resp, err := client.Get(ds.url)
			if err == nil {
				resp.Body.Close()
				return nil
			}
		}
	}
}

func (ds *DevServer) Stop() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.process != nil && ds.process.Process != nil {
		ds.logger.Info("%s", domain.Msg("devserver_stop"))
		_ = ds.process.Process.Signal(os.Interrupt) // nosemgrep: lod-excessive-dot-chain [permanent]
		done := make(chan struct{})
		go func() {
			_ = ds.process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = ds.process.Process.Kill() // nosemgrep: lod-excessive-dot-chain [permanent]
		}
		ds.running = false
	}
}
