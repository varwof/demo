package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func handleListRows(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	where := q.Get("where")
	order := q.Get("order")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	data, err := queryRows(name, where, order, limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, data)
}

func handleInsertRow(w http.ResponseWriter, r *http.Request) {
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

	var req InsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(req.Values) == 0 {
		writeErr(w, http.StatusBadRequest, "values required")
		return
	}

	id, err := insertRow(name, req.Values)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonOK(w, map[string]int64{"inserted": id})
}

func handleUpdateRow(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	pkStr := r.PathValue("pk")
	if name == "" || pkStr == "" {
		writeErr(w, http.StatusBadRequest, "table name and pk required")
		return
	}
	pk, err := strconv.ParseInt(pkStr, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid pk: "+err.Error())
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(req.Values) == 0 {
		writeErr(w, http.StatusBadRequest, "values required")
		return
	}

	affected, err := updateRow(name, pk, req.Values)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonOK(w, map[string]int64{"affected": affected})
}

func handleDeleteRow(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	pkStr := r.PathValue("pk")
	if name == "" || pkStr == "" {
		writeErr(w, http.StatusBadRequest, "table name and pk required")
		return
	}
	pk, err := strconv.ParseInt(pkStr, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid pk: "+err.Error())
		return
	}

	affected, err := deleteRow(name, pk)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonOK(w, map[string]int64{"deleted": affected})
}
