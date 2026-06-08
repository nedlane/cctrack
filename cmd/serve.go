package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/nedlane/cctrack/internal/api"
	"github.com/nedlane/cctrack/internal/config"
	"github.com/nedlane/cctrack/internal/hub"
	"github.com/nedlane/cctrack/internal/parser"
	"github.com/nedlane/cctrack/internal/store"
	"github.com/nedlane/cctrack/internal/tailnet"
	"github.com/nedlane/cctrack/internal/watcher"
	"github.com/spf13/cobra"
)

// WebFSFunc is set by main.go to provide the embedded web filesystem.
var WebFSFunc func() (fs.FS, error)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the dashboard server",
	Long:  "Parse logs, start the web dashboard, and watch for new activity.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Open store
		s, err := store.Open(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer s.Close()

		// Initial parse
		p := parser.New(s)
		files, sessions, err := p.ParseAll(cfg.LogDir)
		if err != nil {
			log.Printf("Warning: initial parse failed: %v", err)
		} else {
			log.Printf("Parsed %d files, %d sessions", files, sessions)
		}

		// Start WebSocket hub
		h := hub.New()
		h.Start()
		defer h.Stop()

		// Start watcher
		w, err := watcher.New(cfg.LogDir, 250*time.Millisecond, func(paths []string) {
			affected, err := p.ParseFiles(paths)
			if err != nil {
				log.Printf("Watcher parse error: %v", err)
				return
			}
			if len(affected) > 0 {
				// Broadcast updates
				for _, sid := range affected {
					sess, err := s.GetSession(sid)
					if err == nil {
						payload, _ := json.Marshal(sess)
						h.Broadcast("session.updated", payload)
					}
				}
				// Broadcast summary update
				summary, err := s.GetSummary()
				if err == nil {
					payload, _ := json.Marshal(summary)
					h.Broadcast("summary.updated", payload)
				}
			}
		})
		if err != nil {
			log.Printf("Warning: file watcher failed to start: %v", err)
		} else {
			w.Start()
			defer w.Stop()
		}

		// Setup HTTP server
		if WebFSFunc == nil {
			return fmt.Errorf("web filesystem not initialized")
		}
		webFS, err := WebFSFunc()
		if err != nil {
			return fmt.Errorf("loading embedded web assets: %w", err)
		}

		mux := http.NewServeMux()
		apiHandler := api.New(s, h, cfg)
		apiHandler.RegisterRoutes(mux)
		mux.Handle("/", api.SPAHandler(webFS))

		addr := fmt.Sprintf(":%d", cfg.Port)
		srv := &http.Server{Addr: addr, Handler: mux}

		// Open browser
		if cfg.OpenBrowserOnServe {
			go func() {
				time.Sleep(200 * time.Millisecond)
				openBrowser(fmt.Sprintf("http://localhost:%d", cfg.Port))
			}()
		}

		// Graceful shutdown
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		// Periodic tailnet sync: pull other machines' logs on an interval and
		// push a summary update so the dashboard reflects the whole tailnet.
		if cfg.Tailnet.Enabled {
			startTailnetSync(ctx, cfg, p, s, h)
		}

		go func() {
			<-ctx.Done()
			log.Println("Shutting down...")
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(shutCtx)
		}()

		log.Printf("Dashboard: http://localhost:%d", cfg.Port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
		return nil
	},
}

// startTailnetSync runs an initial tailnet sync shortly after startup and then
// repeats on the configured interval until ctx is cancelled. After each cycle
// that changed anything, it broadcasts a summary update over the hub.
func startTailnetSync(ctx context.Context, cfg *config.Config, p *parser.Parser, s *store.Store, h *hub.Hub) {
	interval := time.Duration(cfg.Tailnet.SyncIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	syncer := tailnet.FromConfig(cfg, p)

	runOnce := func() {
		report, err := syncer.Sync()
		if err != nil {
			log.Printf("tailnet sync error: %v", err)
			return
		}
		if report.Skipped || report.TotalSessionsAffected == 0 {
			return
		}
		log.Printf("tailnet sync: %d host(s), %d sessions updated", len(report.Hosts), report.TotalSessionsAffected)
		if summary, err := s.GetSummary(); err == nil {
			payload, _ := json.Marshal(summary)
			h.Broadcast("summary.updated", payload)
		}
	}

	go func() {
		// Small initial delay so startup logging stays readable.
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
		runOnce()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runOnce()
			}
		}
	}()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Start()
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
