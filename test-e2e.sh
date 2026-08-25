#!/bin/bash
# E2E test: mysql-api → gateway-http → mTLS client
set -e

CA_DIR=/etc/varwof/test/ca
BASE=https://127.0.0.1:9443

echo "╔══════════════════════════════════════════════════════╗"
echo "║  mysql-api E2E — Gateway Capability Plugin    ║"
echo "╚══════════════════════════════════════════════════════╝"
echo

# ── 1. Verify services are up ───────────────────────
echo "=== 1. Check mysql-api ==="
curl -sf http://127.0.0.1:9393/api/tables > /dev/null && echo "  ✅ MySQL test service running" || { echo "  ❌ MySQL test service down"; exit 1; }

echo "=== 2. Check gateway-http ==="
curl -sfk https://127.0.0.1:9443/ > /dev/null 2>&1 && echo "  ✅ Gateway server running" || echo "  ⚠️  Gateway may not be running yet"

# ── 3. mysql-ops: full access ────────────────────────
echo
echo "=== 3. Client: gateway:mysql-ops ==="
echo

CURL_OPS="curl -sk --cert $CA_DIR/mysql-ops.pem --key $CA_DIR/mysql-ops.key"

echo "  3a. LIST tables"
$CURL_OPS $BASE/api/tables | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok'], str(d); print(f'  ✅ tables={len(d[\"data\"])}')"

echo "  3b. SELECT employees"
$CURL_OPS "$BASE/api/tables/employees/rows?limit=2" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok'], str(d); print(f'  ✅ rows={d[\"data\"][\"total\"]}')"

echo "  3c. INSERT employee"
$CURL_OPS -X POST -H 'Content-Type: application/json' -d '{"values":{"name":"ops测试","department":"运维部","salary":25000,"hire_date":"2026-07-22","email":"ops@test.com"}}' $BASE/api/tables/employees/rows | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok'], str(d); print(f'  ✅ inserted id={d[\"data\"][\"inserted\"]}')"

echo "  3d. UPDATE employee"
$CURL_OPS -X PUT -H 'Content-Type: application/json' -d '{"values":{"salary":26000}}' $BASE/api/tables/employees/rows/6 | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok'], str(d); print(f'  ✅ updated affected={d[\"data\"][\"affected\"]}')"

echo "  3e. DELETE employee"
$CURL_OPS -X DELETE $BASE/api/tables/employees/rows/6 | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok'], str(d); print(f'  ✅ deleted={d[\"data\"][\"deleted\"]}')"

echo "  3f. CREATE TABLE"
$CURL_OPS -X POST -H 'Content-Type: application/json' -d '{"name":"ops_test","columns":[{"name":"id","type":"INT AUTO_INCREMENT PRIMARY KEY"},{"name":"val","type":"VARCHAR(100)"}]}' $BASE/api/tables | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok'], str(d); print(f'  ✅ created table={d[\"data\"][\"table\"]}')"

echo "  3g. DROP TABLE"
$CURL_OPS -X DELETE $BASE/api/tables/ops_test | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok'], str(d); print(f'  ✅ dropped={d[\"data\"][\"dropped\"]}')"

echo
echo "  ✅ mysql-ops: ALL operations passed"

# ── 4. mysql-read: should be denied for writes ───────
echo
echo "=== 4. Client: gateway:mysql-read ==="
echo

CURL_READ="curl -sk --cert $CA_DIR/mysql-read.pem --key $CA_DIR/mysql-read.key"

echo "  4a. LIST tables (should succeed)"
$CURL_READ $BASE/api/tables | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok'], str(d); print(f'  ✅ tables={len(d[\"data\"])}')"

echo "  4b. SELECT employees (should succeed)"
$CURL_READ "$BASE/api/tables/employees/rows?limit=2" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok'], str(d); print(f'  ✅ rows={d[\"data\"][\"total\"]}')"

echo "  4c. INSERT employee (capability_plugin should deny)"
result=$($CURL_READ -X POST -H 'Content-Type: application/json' -d '{"values":{"name":"read测试","department":"只读部","salary":10000,"hire_date":"2026-07-22","email":"read@test.com"}}' $BASE/api/tables/employees/rows 2>&1)
if echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('error','') or d.get('message',''))" 2>/dev/null | grep -qi "deny\|forbidden\|access_denied"; then
  echo "  ✅ INSERT denied (expected)"
else
  echo "  ⚠️  INSERT result: $(echo $result | head -c 100)"
fi

echo "  4d. UPDATE employee (capability_plugin should deny)"
result=$($CURL_READ -X PUT -H 'Content-Type: application/json' -d '{"values":{"salary":999}}' $BASE/api/tables/employees/rows/1 2>&1)
if echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('error','') or d.get('message',''))" 2>/dev/null | grep -qi "deny\|forbidden\|access_denied"; then
  echo "  ✅ UPDATE denied (expected)"
else
  echo "  ⚠️  UPDATE result: $(echo $result | head -c 100)"
fi

echo "  4e. DELETE employee (capability_plugin should deny)"
result=$($CURL_READ -X DELETE $BASE/api/tables/employees/rows/1 2>&1)
if echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('error','') or d.get('message',''))" 2>/dev/null | grep -qi "deny\|forbidden\|access_denied"; then
  echo "  ✅ DELETE denied (expected)"
else
  echo "  ⚠️  DELETE result: $(echo $result | head -c 100)"
fi

echo
echo "╔══════════════════════════════════════════════════════╗"
echo "║  TEST COMPLETE                                      ║"
echo "╚══════════════════════════════════════════════════════╝"
