package main

type ColumnDef struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type TableDef struct {
	Name    string      `json:"name"`
	Columns []ColumnDef `json:"columns"`
}

type CreateTableRequest struct {
	Name    string      `json:"name"`
	Columns []ColumnDef `json:"columns"`
}

type AlterTableRequest struct {
	Action      string     `json:"action"` // add, drop, modify
	ColumnDefs  []ColumnDef `json:"column_defs,omitempty"`
	DropColumns []string   `json:"drop_columns,omitempty"`
}

type InsertRequest struct {
	Values map[string]any `json:"values"`
}

type UpdateRequest struct {
	Values map[string]any `json:"values"`
	Where  string         `json:"where,omitempty"`
}

type RowData struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Total   int              `json:"total,omitempty"`
}

type APIResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type TableInfo struct {
	Name    string `json:"name"`
	Engine  string `json:"engine,omitempty"`
	Rows    int64  `json:"rows,omitempty"`
	Comment string `json:"comment,omitempty"`
}
