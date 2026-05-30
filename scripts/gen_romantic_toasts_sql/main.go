//go:build ignore

// 从 scripts/toasts_raw.ts 生成 sql/romantic_toasts.sql
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}
	root := wd
	for _, name := range []string{"go.mod", "scripts/toasts_raw.ts"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			root = filepath.Dir(root)
			continue
		}
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		fmt.Fprintf(os.Stderr, "cannot find project root from %s\n", wd)
		os.Exit(1)
	}
	rawPath := filepath.Join(root, "scripts", "toasts_raw.ts")
	outPath := filepath.Join(root, "sql", "romantic_toasts.sql")

	raw, err := os.ReadFile(rawPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read raw: %v\n", err)
		os.Exit(1)
	}

	re := regexp.MustCompile(`'([^']*)'`)
	matches := re.FindAllStringSubmatch(string(raw), -1)
	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, "no toasts parsed")
		os.Exit(1)
	}

	var b strings.Builder
	b.WriteString("-- 首页首次访问顶部 toast 浪漫文案池\n")
	b.WriteString("-- PostgreSQL：建表 + 初始数据（可重复执行）\n\n")
	b.WriteString("CREATE TABLE IF NOT EXISTS romantic_toasts (\n")
	b.WriteString("    id         SERIAL PRIMARY KEY,\n")
	b.WriteString("    content    TEXT NOT NULL,\n")
	b.WriteString("    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n")
	b.WriteString(");\n\n")
	b.WriteString("TRUNCATE TABLE romantic_toasts RESTART IDENTITY;\n\n")

	for i, m := range matches {
		text := strings.ReplaceAll(m[1], "''", "'")
		escaped := strings.ReplaceAll(text, "'", "''")
		fmt.Fprintf(&b, "INSERT INTO romantic_toasts (id, content) VALUES (%d, '%s');\n", i+1, escaped)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d rows to %s\n", len(matches), outPath)
}
