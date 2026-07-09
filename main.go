package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
)

// Set by the master on each child to hand off the worker role and index.
const workerEnv = "SIGCAT_WORKER_ID"

// Injected at build time by goreleaser via -ldflags -X.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// worker forms the process tree under the master. It does not read the
// config file: on SIGHUP it only reports that it was signaled. SIGINT and
// SIGTERM are ignored so that no stray signal can thin the pool, which is
// why the master needs no respawn loop to hold the worker count. The master
// ends a worker with SIGKILL.
func runWorker(id int) {
	log.Printf("worker %d started (pid=%d)", id, os.Getpid())

	signal.Ignore(syscall.SIGINT, syscall.SIGTERM)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	for range sigChan {
		log.Printf("worker %d received SIGHUP", id)
	}
}

type Master struct {
	configFile string
	workers    int
	children   []*exec.Cmd
}

func NewMaster(configFile string, workers int) *Master {
	return &Master{
		configFile: configFile,
		workers:    workers,
	}
}

func (m *Master) loadConfig() {
	log.Printf("master loading config from: %s", m.configFile)

	content, err := os.ReadFile(m.configFile)
	if err != nil {
		log.Printf("failed to read config file: %v", err)
		return
	}

	log.Printf("config loaded successfully (%d bytes)", len(content))
	printContent(string(content))
}

func printContent(content string) {
	fmt.Println("=== Config Content ===")
	fmt.Print(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		fmt.Println()
	}
	fmt.Println("=== End of Config ===")
}

func (m *Master) spawnWorkers() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate executable: %w", err)
	}

	for i := 0; i < m.workers; i++ {
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%d", workerEnv, i))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		// Tie each worker to the master's life: if the master dies by any
		// means, including SIGKILL, the kernel sends the worker SIGKILL so
		// no orphan is left behind. Valid because main locks its OS thread.
		cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start worker %d: %w", i, err)
		}
		m.children = append(m.children, cmd)
	}
	return nil
}

// shutdown stops the workers. They ignore SIGTERM, so the master ends them
// with SIGKILL and reaps them to avoid leaving zombies.
func (m *Master) shutdown() {
	for _, cmd := range m.children {
		cmd.Process.Kill()
	}
	for _, cmd := range m.children {
		cmd.Wait()
	}
}

func (m *Master) run() {
	log.Printf("master started (pid=%d), spawning %d workers", os.Getpid(), m.workers)

	m.loadConfig()

	if err := m.spawnWorkers(); err != nil {
		log.Printf("failed to spawn workers: %v", err)
		m.shutdown()
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	log.Println("master ready. Send SIGHUP to reload config, SIGINT/SIGTERM to quit.")

	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGHUP:
			log.Println("master received SIGHUP, reloading config...")
			m.loadConfig()
		case syscall.SIGINT, syscall.SIGTERM:
			log.Printf("master received %v, shutting down...", sig)
			m.shutdown()
			return
		}
	}
}

func main() {
	configFile := flag.String("file", "config.txt", "path to the config file")
	workers := flag.Int("workers", 4, "number of worker processes")
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("sigcat %s\ncommit: %s\ndate: %s\n", version, commit, date)
		return
	}

	if v := os.Getenv(workerEnv); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid %s: %q", workerEnv, v)
		}
		runWorker(id)
		return
	}

	// Pin the master to its OS thread so the thread that forks the workers
	// stays alive for the master's lifetime, keeping their Pdeathsig valid.
	runtime.LockOSThread()

	master := NewMaster(*configFile, *workers)
	master.run()

	log.Println("master stopped")
}
