package paintress

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type DevServer struct {
	cmd     string
	url     string
	dir     string
	logPath string

	mu      sync.Mutex
	process *exec.Cmd
	running bool
}

func NewDevServer(cmd, url, dir, logPath string) *DevServer {
	return &DevServer{cmd: cmd, url: url, dir: dir, logPath: logPath}
}

func (ds *DevServer) Start(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "devserver.start",
		trace.WithAttributes(
			attribute.String("cmd", ds.cmd),
			attribute.String("url", ds.url),
		),
	)
	defer span.End()

	ds.mu.Lock()
	if ds.running {
		ds.mu.Unlock()
		span.AddEvent("devserver.ready", trace.WithAttributes(attribute.Bool("external", false)))
		return nil
	}
	ds.mu.Unlock()

	// Check if dev server is already running externally
	client := &http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get(ds.url); err == nil {
		resp.Body.Close()
		ds.mu.Lock()
		ds.running = true
		ds.mu.Unlock()
		span.AddEvent("devserver.ready", trace.WithAttributes(attribute.Bool("external", true)))
		LogOK("%s", fmt.Sprintf(Msg("devserver_already"), ds.url))
		return nil
	}

	LogInfo("%s", fmt.Sprintf(Msg("devserver_start"), ds.cmd, ds.dir))
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
		ds.mu.Lock()
		ds.running = false
		ds.mu.Unlock()
	}()

	if err := ds.waitReady(ctx); err != nil {
		ds.Stop()
		return err
	}

	ds.mu.Lock()
	ds.running = true
	ds.mu.Unlock()
	span.AddEvent("devserver.ready", trace.WithAttributes(attribute.Bool("external", false)))
	LogOK("%s", fmt.Sprintf(Msg("devserver_ready"), ds.url))
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
			return fmt.Errorf("%s", Msg("devserver_timeout"))
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
		LogInfo("%s", Msg("devserver_stop"))
		_ = ds.process.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() {
			_ = ds.process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = ds.process.Process.Kill()
		}
		ds.running = false
	}
}
