package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	_ "erddiagram/internal/db/extractors"
	"erddiagram/internal/logger"

	"erddiagram/internal/db"
	"erddiagram/internal/introspect"
	"erddiagram/pkg/config"
)

var (
	activeMu      sync.RWMutex
	activeDriver  string
	activeDSN     string
	activeTimeout = 10
	defaultPort   = 8080
)

// setActive sets the active database connection
func setActive(driver, dsn string, timeout int) {
	activeMu.Lock()
	defer activeMu.Unlock()
	activeDriver = driver
	activeDSN = dsn
	activeTimeout = timeout
}

// getActive returns the active databse connection
func getActive() (string, string, int) {
	activeMu.RLock()
	defer activeMu.RUnlock()
	return activeDriver, activeDSN, activeTimeout
}

func main() {
	// flags
	cfgPath := flag.String("config", filepath.Join(".", "configs", "example.yaml"), "path to config YAML")
	driverFlag := flag.String("driver", "", "db driver override (postgres,mysql,sqlite,sqlserver,godror)")
	dsnFlag := flag.String("dsn", "", "dsn override")
	port := flag.Int("port", 0, "http port (overrides config, default"+fmt.Sprintf(" %d)", defaultPort))
	timeout := flag.Int("timeout", 10, "db connect timeout seconds")
	webdir := flag.String("web", filepath.Join(".", "web"), "web ui directory")
	flag.Parse()
	logger.Info("dsnFlag = %s", *dsnFlag)

	// attempt to load config file (optional)
	var appCfg config.AppConfig
	if cfgPath != nil {
		logger.Info("config file %s", *cfgPath)
		if c, err := config.LoadFile(*cfgPath); err == nil {
			appCfg = c
		} else {
			logger.Error("error reading config file: %v", err)
		}
	}

	// allow CLI overrides
	if *driverFlag != "" && *dsnFlag != "" {
		setActive(*driverFlag, *dsnFlag, *timeout)
		appCfg.Database.Type = *driverFlag
		appCfg.Database.DSN = *dsnFlag
		appCfg.Database.Host = ""
		appCfg.Database.Port = 0
		appCfg.Database.Username = ""
		appCfg.Database.Password = ""
		appCfg.Database.DatabaseName = ""
	} else if appCfg.Database.Type != "" {
		drv, dsn, err := config.BuildDriverAndDSN(appCfg.Database)
		if err == nil {
			setActive(drv, dsn, *timeout)
		} else {
			logger.Error("error building DSN: %v", err)
		}
	}

	*port = cmp.Or(*port, appCfg.Server.Port, defaultPort)

	// static web
	fs := http.FileServer(http.Dir(*webdir))
	http.Handle("/", fs)

	// connect endpoint: user requests DB params
	http.HandleFunc("/api/getConnect", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			OK     bool            `json:"ok"`
			Config config.DBConfig `json:"config"`
		}{OK: true, Config: config.DBConfig{
			Type:         config.NormalizeDriver(appCfg.Database.Type),
			DSN:          appCfg.Database.DSN,
			Host:         appCfg.Database.Host,
			Port:         appCfg.Database.Port,
			Username:     appCfg.Database.Username,
			Password:     appCfg.Database.Password,
			DatabaseName: appCfg.Database.DatabaseName,
		}})
	})

	// connect endpoint: user posts DB params to create/test connection
	http.HandleFunc("/api/connect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var dbReq config.DBConfig
		if err := json.NewDecoder(r.Body).Decode(&dbReq); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		appCfg.Database = dbReq
		driver, dsn, err := config.BuildDriverAndDSN(appCfg.Database)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// test connection and return schema on success
		schema, err := db.ConnectAndExtract(driver, dsn, *timeout)
		if err != nil {
			http.Error(w, "connection failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// persist active connection
		setActive(driver, dsn, *timeout)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			OK     bool              `json:"ok"`
			Schema introspect.Schema `json:"schema"`
		}{OK: true, Schema: schema})
	})

	// schema endpoint uses active in-memory connection
	http.HandleFunc("/api/schema", func(w http.ResponseWriter, r *http.Request) {
		driver, dsn, to := getActive()
		if driver == "" || dsn == "" {
			http.Error(w, "no active connection; POST /api/connect to create one", http.StatusBadRequest)
			return
		}
		schema, err := db.ConnectAndExtract(driver, dsn, to)
		if err != nil {
			http.Error(w, "failed to extract schema: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(schema)
	})

	// HTTP server
	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	logger.Info("listening on %s, serving %s", addr, *webdir)
	logger.Info("registered dialects: %v", db.RegisteredDialects())
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatal("%v", err)
	}

}
