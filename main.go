package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"net/http"
	_ "net/http/pprof"

	log "github.com/sirupsen/logrus"
)

var (
	version = "0.0.0"
)

func main() {
	cfgFile := flag.String("config", "", "Path to a config file")
	cfgContent := flag.String("config-content", "", "Alternative to 'config' flag (mutually exclusive). Content of Yaml file that contains configuration.")
	flag.Parse()

	if *cfgFile == "" && *cfgContent == "" {
		log.Fatalf("Config required")
	}

	var cfg *config
	var err error
	if *cfgFile != "" {
		cfg, err = configLoad(*cfgFile)
	} else {
		cfg, err = configParse([]byte(*cfgContent))
	}

	if err != nil {
		log.Fatal(err)
	}

	if cfg.ListenPprof != "" {
		go func() {
			if err := http.ListenAndServe(cfg.ListenPprof, nil); err != nil {
				log.Fatalf("Unable to listen on %s: %s", cfg.ListenPprof, err)
			}
		}()
	}

	if cfg.LogLevel != "" {
		lvl, err := log.ParseLevel(cfg.LogLevel)
		if err != nil {
			log.Fatalf("Unable to parse log level: %s", err)
		}

		log.SetLevel(lvl)
	}

	proc := newProcessor(*cfg)

	if err = proc.run(); err != nil {
		log.Fatalf("Unable to start: %s", err)
	}

	log.Warnf("Listening on %s, sending to %s", cfg.Listen, cfg.Target)
	log.Warnf("Started v%s", version)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, os.Interrupt)
	<-ch

	log.Warn("Shutting down, draining requests")
	if err = proc.close(); err != nil {
		log.Errorf("Error during shutdown: %s", err)
	}

	log.Warnf("Finished")
}
