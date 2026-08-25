package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

func handleListTables(w http.ResponseWriter, r *http.Request) {
	tables, err := listTables()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, tables)
}

func handleCreateTable(w http.ResponseWriter, r *http.Request) {
	var req CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	if len(req.Columns) == 0 {
		writeErr(w, http.StatusBadRequest, "at least one column required")
		return
	}

	var defs []string
	for _, c := range req.Columns {
		if c.Name == "" || c.Type == "" {
			writeErr(w, http.StatusBadRequest, "each column needs name and type")
			return
		}
		defs = append(defs, fmt.Sprintf("`%s` %s", c.Name, c.Type))
	}

	q := fmt.Sprintf("CREATE TABLE `%s` (%s) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", req.Name, strings.Join(defs, ", "))
	if _, err := db.Exec(q); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	slog.Info("table created", "name", req.Name, "columns", len(defs))
	jsonOK(w, map[string]any{"table": req.Name, "columns": len(defs)})
}

func handleDropTable(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "table name required")
		return
	}
	exists, err := tableExists(name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !exists {
		writeErr(w, http.StatusNotFound, fmt.Sprintf("table %q not found", name))
		return
	}
	if _, err := db.Exec(fmt.Sprintf("DROP TABLE `%s`", name)); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("table dropped", "name", name)
	jsonOK(w, map[string]string{"dropped": name})
}

func handleAlterTable(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "table name required")
		return
	}

	var req AlterTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	var clauses []string
	switch req.Action {
	case "add":
		for _, c := range req.ColumnDefs {
			clauses = append(clauses, fmt.Sprintf("ADD COLUMN `%s` %s", c.Name, c.Type))
		}
	case "drop":
		for _, c := range req.DropColumns {
			clauses = append(clauses, fmt.Sprintf("DROP COLUMN `%s`", c))
		}
	case "modify":
		for _, c := range req.ColumnDefs {
			clauses = append(clauses, fmt.Sprintf("MODIFY COLUMN `%s` %s", c.Name, c.Type))
		}
	default:
		writeErr(w, http.StatusBadRequest, "action must be add/drop/modify")
		return
	}

	if len(clauses) == 0 {
		writeErr(w, http.StatusBadRequest, "no columns specified")
		return
	}

	q := fmt.Sprintf("ALTER TABLE `%s` %s", name, strings.Join(clauses, ", "))
	if _, err := db.Exec(q); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("table altered", "name", name, "action", req.Action)
	jsonOK(w, map[string]any{"table": name, "action": req.Action})
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	if err := initSchema(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, map[string]string{"status": "reset_complete"})
}

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(APIResponse{OK: true, Data: data})
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{OK: false, Error: msg})
}
