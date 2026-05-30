#!/usr/bin/env python3
"""Generate sql/romantic_toasts.sql from scripts/toasts_raw.ts."""

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RAW = ROOT / "scripts" / "toasts_raw.ts"
OUT = ROOT / "sql" / "romantic_toasts.sql"


def esc(s: str) -> str:
    return s.replace("'", "''")


def main() -> None:
    text = RAW.read_text(encoding="utf-8")
    toasts = re.findall(r"'((?:\\'|[^'])*)'", text)
    if not toasts:
        raise SystemExit("no toasts parsed")

    lines = [
        "-- 首页首次访问顶部 toast 浪漫文案池",
        "-- PostgreSQL：建表 + 初始数据（可重复执行）",
        "",
        "CREATE TABLE IF NOT EXISTS romantic_toasts (",
        "    id         SERIAL PRIMARY KEY,",
        "    content    TEXT NOT NULL,",
        "    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()",
        ");",
        "",
        "TRUNCATE TABLE romantic_toasts RESTART IDENTITY;",
        "",
    ]
    for i, row in enumerate(toasts, start=1):
        lines.append(f"INSERT INTO romantic_toasts (id, content) VALUES ({i}, '{esc(row)}');")
    lines.append("")
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text("\n".join(lines), encoding="utf-8")
    print(f"wrote {len(toasts)} rows to {OUT}")


if __name__ == "__main__":
    main()
