#!/bin/bash
# mysql-api API smoke test
# Usage: bash test-api.sh [base_url]

BASE="${1:-http://127.0.0.1:9393}"
PASS=0
FAIL=0

ok()   { PASS=$((PASS+1)); echo "  ✅ $1"; }
fail() { FAIL=$((FAIL+1)); echo "  ❌ $1"; }

check() {
  local desc="$1" method="$2" url="$3" data="$4" expect="$5"
  local out
  if [ -n "$data" ]; then
    out=$(curl -s -X "$method" -H 'Content-Type: application/json' -d "$data" "${BASE}${url}")
  else
    out=$(curl -s -X "$method" "${BASE}${url}")
  fi
  if echo "$out" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d.get('ok') == $expect, f'got {d}'" 2>/dev/null; then
    ok "$desc"
  else
    fail "$desc (response: $(echo $out | head -c 120))"
  fi
}

echo "=== mysql-api smoke test ==="
echo "base: $BASE"
echo

# ── Table management ──────────────────────────────
check "LIST TABLES"            GET  /api/tables                            ''   True
check "CREATE TABLE projects"  POST /api/tables                            '{"name":"projects","columns":[{"name":"id","type":"INT AUTO_INCREMENT PRIMARY KEY"},{"name":"name","type":"VARCHAR(100) NOT NULL"},{"name":"status","type":"VARCHAR(20) DEFAULT \"active\""}]}' True
check "ALTER TABLE (add col)"  POST '/api/tables/projects/alter'           '{"action":"add","column_defs":[{"name":"budget","type":"DECIMAL(12,2)"}]}' True

# ── Row CRUD ──────────────────────────────────────
check "INSERT employee"        POST /api/tables/employees/rows             '{"values":{"name":"测试员","department":"研发部","salary":20000,"hire_date":"2026-07-22","email":"test@example.com"}}' True
check "UPDATE employee id=6"   PUT  '/api/tables/employees/rows/6'         '{"values":{"salary":22000,"department":"测试部"}}' True
check "DELETE employee id=6"   DELETE '/api/tables/employees/rows/6'       ''   True
check "INSERT project row"     POST /api/tables/projects/rows              '{"values":{"name":"网关v2重构","budget":500000}}' True
check "UPDATE project id=1"    PUT  '/api/tables/projects/rows/1'          '{"values":{"status":"completed"}}' True
check "DELETE project row"     DELETE '/api/tables/projects/rows/1'        ''   True
check "LIST employees"         GET  '/api/tables/employees/rows?limit=3'   ''   True
check "LIST products"          GET  '/api/tables/products/rows?limit=3'    ''   True
check "LIST orders"            GET  '/api/tables/orders/rows?limit=3'      ''   True

# ── Edge cases ────────────────────────────────────
check "DELETE non-existent pk" DELETE '/api/tables/employees/rows/999'     ''   True
check "DROP non-existent table" DELETE '/api/tables/nonexist'              ''   False
check "CREATE duplicate table" POST /api/tables                            '{"name":"employees","columns":[{"name":"x","type":"INT"}]}' False
check "INSERT empty values"    POST /api/tables/employees/rows             '{}' False

# ── Cleanup ───────────────────────────────────────
check "DROP TABLE projects"    DELETE '/api/tables/projects'               ''   True
check "RESET schema"           POST /api/reset                             ''   True
check "VERIFY reset (5 employees)" GET '/api/tables/employees/rows?limit=5' '' True

echo
echo "=== Result: $PASS passed, $FAIL failed ==="
exit $FAIL
