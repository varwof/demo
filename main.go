package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	cfgPath := flag.String("config", "/etc/varwof/test/mysql/config.json", "config file path")
	listen := flag.String("listen", "", "HTTP listen address (overrides config)")
	dsn := flag.String("dsn", "", "MySQL DSN (overrides config)")
	reset := flag.Bool("reset", false, "drop and recreate all test tables on startup")
	flag.Parse()

	appCfg, err := LoadAppConfig(*cfgPath)
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	if *listen != "" {
		appCfg.Listen = *listen
	}
	if *dsn != "" {
		appCfg.DSN = *dsn
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	slog.Info("starting mysql-api", "listen", appCfg.Listen, "config", *cfgPath)

	if err := openDB(appCfg.DSN); err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("connected to MySQL", "database", "pkitest")

	if *reset {
		slog.Info("resetting schema")
		if err := initSchema(); err != nil {
			slog.Error("init schema failed", "error", err)
			os.Exit(1)
		}
	} else {
		exists, err := tableExists("employees")
		if err != nil {
			slog.Error("check employees table", "error", err)
			os.Exit(1)
		}
		if !exists {
			slog.Info("initializing schema for first run")
			if err := initSchema(); err != nil {
				slog.Error("init schema failed", "error", err)
				os.Exit(1)
			}
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/tables", handleListTables)
	mux.HandleFunc("POST /api/tables", handleCreateTable)
	mux.HandleFunc("DELETE /api/tables/{name}", handleDropTable)
	mux.HandleFunc("POST /api/tables/{name}/alter", handleAlterTable)
	mux.HandleFunc("POST /api/reset", handleReset)

	mux.HandleFunc("GET /api/tables/{name}/rows", handleListRows)
	mux.HandleFunc("POST /api/tables/{name}/rows", handleInsertRow)
	mux.HandleFunc("PUT /api/tables/{name}/rows/{pk}", handleUpdateRow)
	mux.HandleFunc("DELETE /api/tables/{name}/rows/{pk}", handleDeleteRow)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "mysql-api — MySQL Test Database Manager")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Tables management:")
		fmt.Fprintln(w, "  GET    /api/tables                    — list tables")
		fmt.Fprintln(w, "  POST   /api/tables                    — create table")
		fmt.Fprintln(w, "  DELETE /api/tables/{name}             — drop table")
		fmt.Fprintln(w, "  POST   /api/tables/{name}/alter       — alter table")
		fmt.Fprintln(w, "  POST   /api/reset                     — reset all tables")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Row CRUD:")
		fmt.Fprintln(w, "  GET    /api/tables/{name}/rows        — list rows (?where=&order=&limit=&offset=)")
		fmt.Fprintln(w, "  POST   /api/tables/{name}/rows        — insert row")
		fmt.Fprintln(w, "  PUT    /api/tables/{name}/rows/{pk}   — update row by id")
		fmt.Fprintln(w, "  DELETE /api/tables/{name}/rows/{pk}   — delete row by id")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Demo tables: employees, products, orders")
	})

	slog.Info("listening", "addr", appCfg.Listen)
	if err := http.ListenAndServe(appCfg.Listen, mux); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
