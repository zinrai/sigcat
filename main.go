package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type Daemon struct {
	configFile string
	content    string
	lastError  error
}

func NewDaemon(configFile string) *Daemon {
	return &Daemon{
		configFile: configFile,
	}
}

func (d *Daemon) loadConfig() error {
	log.Printf("Loading config from: %s", d.configFile)

	content, err := ioutil.ReadFile(d.configFile)
	if err != nil {
		d.lastError = err
		return fmt.Errorf("failed to read config file: %w", err)
	}

	d.content = string(content)
	d.lastError = nil

	log.Printf("Config loaded successfully (%d bytes)", len(content))
	return nil
}

func (d *Daemon) printContent() {
	if d.lastError != nil {
		log.Printf("Error: %v", d.lastError)
		return
	}

	fmt.Println("=== Config Content ===")
	fmt.Print(d.content)
	if len(d.content) > 0 && d.content[len(d.content)-1] != '\n' {
		fmt.Println()
	}
	fmt.Println("=== End of Config ===")
}

func (d *Daemon) run() {
	// Initial config load
	if err := d.loadConfig(); err != nil {
		log.Printf("Initial config load failed: %v", err)
	} else {
		d.printContent()
	}

	// Setup signal handlers
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Daemon started. Send SIGHUP to reload config, SIGINT/SIGTERM to quit.")

	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGHUP:
			log.Println("Received SIGHUP, reloading config...")
			if err := d.loadConfig(); err != nil {
				log.Printf("Config reload failed: %v", err)
			} else {
				d.printContent()
			}
		case syscall.SIGINT, syscall.SIGTERM:
			log.Printf("Received %v, shutting down...", sig)
			return
		}
	}
}

func main() {
	configFile := "config.txt"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	daemon := NewDaemon(configFile)
	daemon.run()

	log.Println("Daemon stopped")
}
