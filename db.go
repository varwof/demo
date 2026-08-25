package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func openDB(dsn string) error {
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err = db.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	return nil
}

var seedTables = []struct {
	Name    string
	Columns []string
	Seed    string
}{
	{
		Name: "employees",
		Columns: []string{
			"id INT AUTO_INCREMENT PRIMARY KEY",
			"name VARCHAR(100) NOT NULL",
			"department VARCHAR(50) NOT NULL",
			"salary DECIMAL(10,2) NOT NULL",
			"hire_date DATE NOT NULL",
			"email VARCHAR(100)",
		},
		Seed: `INSERT IGNORE INTO employees (id,name,department,salary,hire_date,email) VALUES
			(1,'张三','技术部',15000.00,'2023-01-15','zhangsan@example.com'),
			(2,'李四','市场部',12000.00,'2023-03-20','lisi@example.com'),
			(3,'王五','技术部',18000.00,'2022-06-01','wangwu@example.com'),
			(4,'赵六','人事部',10000.00,'2024-02-01','zhaoliu@example.com'),
			(5,'钱七','财务部',13000.00,'2023-09-10','qianqi@example.com')`,
	},
	{
		Name: "products",
		Columns: []string{
			"id INT AUTO_INCREMENT PRIMARY KEY",
			"name VARCHAR(100) NOT NULL",
			"category VARCHAR(50) NOT NULL",
			"price DECIMAL(10,2) NOT NULL",
			"stock INT NOT NULL DEFAULT 0",
			"description TEXT",
		},
		Seed: `INSERT IGNORE INTO products (id,name,category,price,stock,description) VALUES
			(1,'轻薄笔记本','电子产品',5999.00,50,'14英寸 16GB内存 512GB固态'),
			(2,'无线鼠标','电子产品',89.00,200,'蓝牙5.0 静音设计'),
			(3,'机械键盘','电子产品',299.00,150,'青轴 87键 RGB背光'),
			(4,'办公桌','家具',1299.00,30,'1.6米 实木颗粒板'),
			(5,'人体工学椅','家具',899.00,45,'全网面 可调节扶手')`,
	},
	{
		Name: "orders",
		Columns: []string{
			"id INT AUTO_INCREMENT PRIMARY KEY",
			"product_id INT NOT NULL",
			"quantity INT NOT NULL",
			"total DECIMAL(10,2) NOT NULL",
			"customer_name VARCHAR(100) NOT NULL",
			"order_date DATETIME NOT NULL",
			"status VARCHAR(20) DEFAULT 'pending'",
		},
		Seed: `INSERT IGNORE INTO orders (id,product_id,quantity,total,customer_name,order_date,status) VALUES
			(1,1,2,11998.00,'张伟','2026-07-01 10:30:00','completed'),
			(2,2,5,445.00,'李娜','2026-07-02 14:00:00','completed'),
			(3,3,1,299.00,'王强','2026-07-03 09:15:00','pending'),
			(4,4,3,3897.00,'刘洋','2026-07-05 16:45:00','shipped'),
			(5,5,2,1798.00,'陈明','2026-07-06 11:20:00','pending')`,
	},
}

func initSchema() error {
	for _, t := range seedTables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", t.Name))
		if err != nil {
			return fmt.Errorf("drop %s: %w", t.Name, err)
		}
		_, err = db.Exec(fmt.Sprintf("CREATE TABLE `%s` (%s) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
			t.Name, strings.Join(t.Columns, ", ")))
		if err != nil {
			return fmt.Errorf("create %s: %w", t.Name, err)
		}
		slog.Info("created table", "name", t.Name)
	}
	for _, t := range seedTables {
		if t.Seed != "" {
			if _, err := db.Exec(t.Seed); err != nil {
				return fmt.Errorf("seed %s: %w", t.Name, err)
			}
			slog.Info("seeded table", "name", t.Name)
		}
	}
	return nil
}

func listTables() ([]TableInfo, error) {
	rows, err := db.Query("SELECT TABLE_NAME, ENGINE, TABLE_ROWS, TABLE_COMMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE()")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name, &t.Engine, &t.Rows, &t.Comment); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, nil
}

func tableExists(name string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", name).Scan(&count)
	return count > 0, err
}

func getColumns(table string) ([]string, error) {
	rows, err := db.Query("SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION", table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, nil
}

func queryRows(table string, where, order string, limit, offset int) (*RowData, error) {
	cols, err := getColumns(table)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf("SELECT * FROM `%s`", table)
	var args []any
	if where != "" {
		q += " WHERE " + where
	}
	if order != "" {
		q += " ORDER BY " + order
	} else if len(cols) > 0 {
		q += " ORDER BY " + cols[0]
	}
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			q += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	slog.Debug("query", "sql", q)

	rrows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rrows.Close()

	var result RowData
	result.Columns = cols

	for rrows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rrows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any)
		for i, c := range cols {
			row[c] = formatValue(vals[i])
		}
		result.Rows = append(result.Rows, row)
	}

	var total int
	if where != "" {
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE %s", table, where)).Scan(&total)
	} else {
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)).Scan(&total)
	}
	result.Total = total

	return &result, nil
}

func insertRow(table string, vals map[string]any) (int64, error) {
	var cols []string
	var ph []string
	var args []any
	for k, v := range vals {
		cols = append(cols, k)
		ph = append(ph, "?")
		args = append(args, v)
	}
	q := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		table, strings.Join(cols, ", "), strings.Join(ph, ", "))
	slog.Debug("insert", "sql", q, "args", args)
	r, err := db.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func updateRow(table string, id int64, vals map[string]any) (int64, error) {
	var sets []string
	var args []any
	for k, v := range vals {
		sets = append(sets, fmt.Sprintf("`%s` = ?", k))
		args = append(args, v)
	}
	args = append(args, id)
	q := fmt.Sprintf("UPDATE `%s` SET %s WHERE id = ?", table, strings.Join(sets, ", "))
	slog.Debug("update", "sql", q, "args", args)
	r, err := db.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

func deleteRow(table string, id int64) (int64, error) {
	q := fmt.Sprintf("DELETE FROM `%s` WHERE id = ?", table)
	r, err := db.Exec(q, id)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

func formatValue(v any) any {
	switch vv := v.(type) {
	case []byte:
		s := string(vv)
		return s
	default:
		return vv
	}
}
