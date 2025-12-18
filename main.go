package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/actionsum/actionsum/internal/config"
	"github.com/actionsum/actionsum/internal/daemon"
	"github.com/actionsum/actionsum/internal/database"
	"github.com/actionsum/actionsum/internal/reporter"
	"github.com/actionsum/actionsum/internal/tracker"
	"github.com/actionsum/actionsum/internal/web"
	"github.com/actionsum/actionsum/pkg/detector"
	"github.com/actionsum/actionsum/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "start":
		startDaemon()
	case "serve":
		serveDaemon()
	case "stop":
		stopDaemon()
	case "status":
		showStatus()
	case "report":
		generateReport()
	case "clear":
		clearDatabase()
	case "version":
		showVersion()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`actionsum - Application focus time tracker

Usage:
  actionsum <command> [options]

Commands:
  start              Start the tracking daemon
  serve              Start daemon with web API server
  stop               Stop the tracking daemon
  status             Show daemon status and current focused app
  report [period]    Generate time report (period: day, week, month)
  clear              Clear all tracking data from database
  version            Show version information
  help               Show this help message

Examples:
  actionsum start
  actionsum serve
  actionsum status
  actionsum report day
  actionsum report week
  actionsum stop

Environment Variables:
  ACTIONSUM_DB_PATH          Database file path
  ACTIONSUM_POLL_INTERVAL    Poll interval in seconds (10-300)
  ACTIONSUM_IDLE_THRESHOLD   Idle threshold in seconds
  ACTIONSUM_PID_FILE         PID file path
  ACTIONSUM_EXCLUDE_IDLE     Exclude idle time from reports (true/false)

Version: %s
`, version.Version)
}

func startDaemon() {
	cfg := config.New()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	dm := daemon.New(cfg.Daemon.PIDFile)
	running, pid, err := dm.IsRunning()
	if err != nil {
		log.Fatalf("Failed to check daemon status: %v", err)
	}
	if running {
		log.Fatalf("Daemon is already running (PID: %d)", pid)
	}

	if os.Getenv("ACTIONSUM_DAEMON_CHILD") != "1" {
		daemonize(false)
		return
	}

	runStartDaemon(cfg, dm)
}

func runStartDaemon(cfg *config.Config, dm *daemon.Daemon) {
	logPath := fmt.Sprintf("/tmp/actionsum-%d.log", os.Getuid())
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	db, err := database.Connect(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	det, err := detector.New()
	if err != nil {
		log.Fatalf("Failed to initialize window detector: %v", err)
	}
	defer det.Close()

	log.Printf("Window detector initialized: %s", det.GetDisplayServer())

	if err := dm.WritePID(); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}
	defer dm.RemovePID()

	repo := database.NewRepository(db)
	trackerSvc := tracker.NewService(cfg, repo, det)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
		trackerSvc.Stop()
	}()

	log.Println("Starting actionsum daemon...")
	log.Printf("Configuration:\n%s", cfg.String())

	if err := trackerSvc.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Tracker error: %v", err)
	}

	log.Println("Daemon stopped successfully")
}

func stopDaemon() {
	cfg := config.New()
	dm := daemon.New(cfg.Daemon.PIDFile)
	running, pid, err := dm.IsRunning()
	if err != nil {
		log.Fatalf("Failed to check daemon status: %v", err)
	}
	if !running {
		fmt.Println("Daemon is not running")
		return
	}
	fmt.Printf("Stopping daemon (PID: %d)...\n", pid)
	if err := dm.Stop(); err != nil {
		log.Fatalf("Failed to stop daemon: %v", err)
	}
	fmt.Println("Daemon stopped successfully")
}

func showStatus() {
	cfg := config.New()
	dm := daemon.New(cfg.Daemon.PIDFile)
	running, pid, err := dm.IsRunning()
	if err != nil {
		log.Fatalf("Failed to check daemon status: %v", err)
	}
	if !running {
		fmt.Println("Status: Not running")
	} else {
		fmt.Printf("Status: Running (PID: %d)\n", pid)
		fmt.Printf("Poll Interval: %v\n", cfg.Tracker.PollInterval)
		fmt.Printf("Database: %s\n", cfg.Database.Path)
	}

	det, err := detector.New()
	if err != nil {
		fmt.Printf("\nCould not detect current window: %v\n", err)
		return
	}
	defer det.Close()

	windowInfo, err := det.GetFocusedWindow()
	if err == nil && windowInfo != nil {
		fmt.Printf("\nCurrent Window:\n")
		fmt.Printf("  App: %s\n", windowInfo.AppName)
		fmt.Printf("  Title: %s\n", windowInfo.WindowTitle)
		fmt.Printf("  Display: %s\n", windowInfo.DisplayServer)
	}

	idleInfo, err := det.GetIdleInfo()
	if err == nil && idleInfo != nil {
		fmt.Printf("\nSystem State:\n")
		fmt.Printf("  Idle: %v\n", idleInfo.IsIdle)
		fmt.Printf("  Locked: %v\n", idleInfo.IsLocked)
		if idleInfo.IdleTime > 0 {
			fmt.Printf("  Idle Time: %ds\n", idleInfo.IdleTime)
		}
	}
}

func generateReport() {
	periodType := "day"
	if len(os.Args) > 2 {
		periodType = os.Args[2]
	}
	cfg := config.New()
	db, err := database.Connect(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	rep := reporter.New(cfg, repo)

	jsonOutput := false
	if len(os.Args) > 3 && os.Args[3] == "--json" {
		jsonOutput = true
	}
	report, err := rep.GenerateReport(periodType)
	if err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}
	if jsonOutput {
		jsonStr, err := rep.FormatReportJSON(report)
		if err != nil {
			log.Fatalf("Failed to format JSON: %v", err)
		}
		fmt.Println(jsonStr)
	} else {
		fmt.Println(rep.FormatReportText(report))
	}
}

func clearDatabase() {
	cfg := config.New()
	fmt.Print("This will delete all tracking data. Are you sure? (yes/no): ")
	var response string
	fmt.Scanln(&response)
	if response != "yes" && response != "y" {
		fmt.Println("Operation cancelled")
		return
	}
	db, err := database.Connect(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	if err := repo.Clear(); err != nil {
		log.Fatalf("Failed to clear database: %v", err)
	}
	fmt.Println("Database cleared successfully")
}

func serveDaemon() {
	cfg := config.New()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	dm := daemon.New(cfg.Daemon.PIDFile)
	running, pid, err := dm.IsRunning()
	if err != nil {
		log.Fatalf("Failed to check daemon status: %v", err)
	}
	if running {
		log.Fatalf("Daemon is already running (PID: %d)", pid)
	}
	if os.Getenv("ACTIONSUM_DAEMON_CHILD") != "1" {
		daemonize(true)
		return
	}
	runServeDaemon(cfg, dm)
}

func runServeDaemon(cfg *config.Config, dm *daemon.Daemon) {
	logPath := fmt.Sprintf("/tmp/actionsum-%d.log", os.Getuid())
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}
	db, err := database.Connect(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	if err := db.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	det, err := detector.New()
	if err != nil {
		log.Fatalf("Failed to initialize window detector: %v", err)
	}
	defer det.Close()
	log.Printf("Window detector initialized: %s", det.GetDisplayServer())
	if err := dm.WritePID(); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}
	defer dm.RemovePID()
	repo := database.NewRepository(db)
	trackerSvc := tracker.NewService(cfg, repo, det)
	webServer := web.NewServer(cfg, repo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := webServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("Web server error: %v", err)
		}
	}()
	go func() {
		if err := trackerSvc.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("Tracker error: %v", err)
			cancel()
		}
	}()
	log.Println("Starting actionsum daemon with web API...")
	log.Printf("Web API available at: http://%s", webServer.GetAddress())
	log.Printf("Configuration:\n%s", cfg.String())
	<-sigChan
	log.Println("Received shutdown signal")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	cancel()
	trackerSvc.Stop()
	if err := webServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down web server: %v", err)
	}
	log.Println("Daemon stopped successfully")
}

func daemonize(withWeb bool) {
	env := os.Environ()
	env = append(env, "ACTIONSUM_DAEMON_CHILD=1")
	args := os.Args
	procAttr := &os.ProcAttr{
		Env:   env,
		Files: []*os.File{nil, nil, nil},
		Sys:   &syscall.SysProcAttr{Setsid: true},
	}
	process, err := os.StartProcess(args[0], args, procAttr)
	if err != nil {
		log.Fatalf("Failed to start daemon process: %v", err)
	}
	logPath := fmt.Sprintf("/tmp/actionsum-%d.log", os.Getuid())
	if withWeb {
		fmt.Printf("Daemon started successfully (PID: %d)\n", process.Pid)
		fmt.Println("Web API available at: http://localhost:8080")
		fmt.Printf("Logs: %s\n", logPath)
	} else {
		fmt.Printf("Daemon started successfully (PID: %d)\n", process.Pid)
		fmt.Printf("Logs: %s\n", logPath)
	}
}

func showVersion() {
	fmt.Printf("version: %s\n", version.Version)
	fmt.Printf("built  : %s\n", version.Date)
}
