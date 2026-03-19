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

// devServerReadyTimeout is the maximum wait for the dev server health check.
var devServerReadyTimeout = 60 * time.Second

// devServerStopTimeout is the grace period before SIGKILL after SIGINT.
var devServerStopTimeout = 5 * time.Second

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
			attribute.String("cmd", platform.SanitizeUTF8(ds.cmd)),
			attribute.String("url", platform.SanitizeUTF8(ds.url)),
		),
	)
	defer span.End()

	parts := strings.Fields(ds.cmd)
	if len(parts) == 0 {
		return fmt.Errorf("dev_cmd is empty; set dev_cmd or enable no_dev")
	}

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
	deadline := time.After(devServerReadyTimeout)
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
		case <-time.After(devServerStopTimeout):
			_ = ds.process.Process.Kill() // nosemgrep: lod-excessive-dot-chain [permanent]
		}
		ds.running = false
	}
}
