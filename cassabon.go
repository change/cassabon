package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jeffpierce/cassabon/config"
	"github.com/jeffpierce/cassabon/logging"
)

func fetchConfiguration(confFile string) {

	// Open, read, and parse the YAML.
	cnf := config.ParseConfig(confFile)

	// Fill in arguments not provided on the command line.
	if config.G.Log.Logdir == "" {
		config.G.Log.Logdir = cnf.Logging.Logdir
	}
	if config.G.Log.Loglevel == "" {
		config.G.Log.Loglevel = cnf.Logging.Loglevel
	}
	if config.G.Statsd.Host == "" {
		config.G.Statsd.Host = cnf.Statsd.Host
	}

	// Copy in values sourced solely from the configuration file.
}

func main() {

	// The name of the YAML configuration file.
	var confFile string

	// Get options provided on the command line.
	flag.StringVar(&confFile, "conf", "", "Location of YAML configuration file.")
	flag.StringVar(&config.G.Log.Logdir, "logdir", "", "Name of directory to contain log files (stderr if unspecified)")
	flag.StringVar(&config.G.Log.Loglevel, "loglevel", "debug", "Log level: debug|info|warn|error|fatal")
	flag.StringVar(&config.G.Statsd.Host, "statsdhost", "", "statsd host or IP address")
	flag.IntVar(&config.G.Statsd.Port, "statsdport", 8125, "statsd port")
	flag.Parse()

	// Fill in startup values not provided on the command line, if available.
	if confFile != "" {
		fetchConfiguration(confFile)
	}

	// Set up logging.
	sev, errLogLevel := logging.TextToSeverity(config.G.Log.Loglevel)
	if config.G.Log.Logdir != "" {
		logDir, _ := filepath.Abs(config.G.Log.Logdir)
		config.G.Log.System = logging.NewLogger("system", filepath.Join(logDir, "cassabon.system.log"), sev)
		config.G.Log.Carbon = logging.NewLogger("carbon", filepath.Join(logDir, "cassabon.carbon.log"), sev)
		config.G.Log.API = logging.NewLogger("api", filepath.Join(logDir, "cassabon.api.log"), sev)
	} else {
		config.G.Log.System = logging.NewLogger("system", "", sev)
		config.G.Log.Carbon = logging.NewLogger("carbon", "", sev)
		config.G.Log.API = logging.NewLogger("api", "", sev)
	}
	defer config.G.Log.System.Close()
	defer config.G.Log.Carbon.Close()
	defer config.G.Log.API.Close()

	// Announce the application startup in the logs.
	config.G.Log.System.LogInfo("Application startup in progress")
	if errLogLevel != nil {
		config.G.Log.System.LogWarn("Bad command line argument: %v", errLogLevel)
	}

	// Set up stats reporting.
	if config.G.Statsd.Host != "" {
		hp := fmt.Sprintf("%s:%d", config.G.Statsd.Host, config.G.Statsd.Port)
		if err := logging.S.Open(hp, "cassabon"); err != nil {
			config.G.Log.System.LogError("Not reporting to statsd: %v", err)
		} else {
			config.G.Log.System.LogInfo("Reporting to statsd at %s", hp)
		}
		defer logging.S.Close()
	} else {
		config.G.Log.System.LogInfo("Not reporting to statsd: specify host or IP to enable")
	}

	// Set up reload and termination signal handlers.
	var sighup = make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	var sigterm = make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	// Repeat until terminated by SIGINT/SIGTERM.
	configIsStale := false
	repeat := true
	for repeat {

		// Perform initialization that is repeated on every SIGHUP.
		config.G.Log.System.LogInfo("Application reading and applying current configuration")
		if configIsStale && confFile != "" {
			fetchConfiguration(confFile)
		}

		// Wait for receipt of a recognized signal.
		config.G.Log.System.LogInfo("Application running")
		select {
		case <-sighup:
			config.G.Log.System.LogInfo("Application received SIGHUP")
			logging.Reopen()
			configIsStale = true
		case <-sigterm:
			config.G.Log.System.LogInfo("Application received SIGINT/SIGTERM, preparing to terminate")
			repeat = false
		}
	}

	// Final cleanup.
	config.G.Log.System.LogInfo("Application termination complete")
}
