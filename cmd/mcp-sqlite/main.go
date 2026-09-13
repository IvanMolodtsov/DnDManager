package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	_ "modernc.org/sqlite"
)

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	dbPath := os.Getenv("DNDMANAGER_DB")
	if dbPath == "" {
		dbPath = filepath.Join("data", "dnd.db")
	}
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		fatal(err)
	}
	if _, err := os.Stat(abs); err != nil {
		fatal(fmt.Errorf("database %s: %w", abs, err))
	}
	dsn := fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(5000)", filepath.ToSlash(abs))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		var req rpcReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			writeErr(nil, -32700, "parse error")
			continue
		}
		handle(db, abs, req)
	}
	if err := in.Err(); err != nil && err != io.EOF {
		fatal(err)
	}
}

func handle(db *sql.DB, abs string, req rpcReq) {
	switch req.Method {
	case "initialize":
		writeResult(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "dndmanager-db", "version": "1.0.0"},
		})
	case "notifications/initialized", "notifications/cancelled":
		return
	case "ping":
		writeResult(req.ID, map[string]any{})
	case "tools/list":
		writeResult(req.ID, map[string]any{"tools": tools()})
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeErr(req.ID, -32602, "bad params")
			return
		}
		text, err := call(db, abs, p.Name, p.Arguments)
		if err != nil {
			writeResult(req.ID, map[string]any{
				"content": []map[string]string{{"type": "text", "text": err.Error()}},
				"isError": true,
			})
			return
		}
		writeResult(req.ID, map[string]any{
			"content": []map[string]string{{"type": "text", "text": text}},
		})
	default:
		if strings.HasPrefix(req.Method, "notifications/") {
			return
		}
		writeErr(req.ID, -32601, "method not found: "+req.Method)
	}
}

func tools() []map[string]any {
	return []map[string]any{
		{
			"name":        "list_tables",
			"description": "List user tables in the local DnDManager SQLite database (data/dnd.db).",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "describe_table",
			"description": "Show columns for one table (PRAGMA table_info).",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{"table": map[string]any{"type": "string"}},
				"required":   []string{"table"},
			},
		},
		{
			"name":        "query",
			"description": "Run a read-only SQL query (SELECT / WITH / PRAGMA table_info). The live campaign database, not a guess from code.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{"sql": map[string]any{"type": "string"}},
				"required":   []string{"sql"},
			},
		},
	}
}

func call(db *sql.DB, abs, name string, args map[string]any) (string, error) {
	switch name {
	case "list_tables":
		return runQuery(db, `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	case "describe_table":
		table, _ := args["table"].(string)
		table = strings.TrimSpace(table)
		if !identOK(table) {
			return "", fmt.Errorf("invalid table name")
		}
		return runQuery(db, "PRAGMA table_info("+quoteIdent(table)+")")
	case "query":
		sqlText, _ := args["sql"].(string)
		if !readOnlySQL(sqlText) {
			return "", fmt.Errorf("only read-only SELECT/WITH/PRAGMA table_info queries are allowed")
		}
		out, err := runQuery(db, sqlText)
		if err != nil {
			return "", err
		}
		return abs + "\n" + out, nil
	default:
		return "", fmt.Errorf("unknown tool %s", name)
	}
}

func runQuery(db *sql.DB, q string) (string, error) {
	rows, err := db.Query(q)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	var out []map[string]any
	for rows.Next() {
		raw := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range raw {
			ptr[i] = &raw[i]
		}
		if err := rows.Scan(ptr...); err != nil {
			return "", err
		}
		row := map[string]any{}
		for i, c := range cols {
			row[c] = cell(raw[i])
		}
		out = append(out, row)
		if len(out) >= 500 {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func cell(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(t)
	default:
		return t
	}
}

func readOnlySQL(q string) bool {
	s := stripSQLComments(q)
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	kw := firstKeyword(s)
	switch kw {
	case "SELECT", "WITH", "PRAGMA":
		upper := strings.ToUpper(s)
		for _, bad := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "ATTACH", "DETACH", "REPLACE", "VACUUM", "REINDEX"} {
			if strings.Contains(upper, bad) && kw != "PRAGMA" {
				// WITH ... SELECT is fine; WITH + INSERT is not. Cheap check:
				if bad != "WITH" && strings.Contains(upper, bad+" ") {
					return false
				}
			}
		}
		if kw == "PRAGMA" {
			rest := strings.ToUpper(strings.TrimSpace(s[6:]))
			return strings.HasPrefix(rest, "TABLE_INFO") || strings.HasPrefix(rest, "TABLE_LIST") || strings.HasPrefix(rest, "FOREIGN_KEY_LIST")
		}
		return true
	default:
		return false
	}
}

func stripSQLComments(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '-' && s[i+1] == '-' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func firstKeyword(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			break
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
			b.WriteRune(unicode.ToUpper(r))
			continue
		}
		break
	}
	return b.String()
}

func identOK(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func writeResult(id json.RawMessage, result any) {
	if len(id) == 0 {
		return
	}
	writeMsg(map[string]any{"jsonrpc": "2.0", "id": jsonRaw(id), "result": result})
}

func writeErr(id json.RawMessage, code int, msg string) {
	m := map[string]any{"jsonrpc": "2.0", "error": rpcErr{Code: code, Message: msg}}
	if len(id) > 0 {
		m["id"] = jsonRaw(id)
	}
	writeMsg(m)
}

func jsonRaw(id json.RawMessage) any {
	var v any
	if err := json.Unmarshal(id, &v); err != nil {
		return nil
	}
	return v
}

func writeMsg(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	os.Stdout.Write(append(b, '\n'))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
