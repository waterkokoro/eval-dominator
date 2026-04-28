package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"eval-dominator/backend/internal/config"
)

func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("SQLite 路径不能为空")
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConnections)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接 SQLite 失败: %w", err)
	}

	return db, nil
}

func Migrate(db *sql.DB, migrationPath string) error {
	// 顺序：先跑 001_init.sql（CREATE TABLE IF NOT EXISTS 建出全部表），再跑 ensureColumn 对老库补列。
	// 不能反过来 —— 否则全新空库时 ensureColumn 找不到表会试图 ALTER 不存在的表而失败。
	data, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("读取数据库迁移脚本失败: %w", err)
	}
	if _, err := db.Exec(string(data)); err != nil {
		return fmt.Errorf("执行数据库迁移失败: %w", err)
	}

	// 老库兼容：SQLite 不支持 ADD COLUMN IF NOT EXISTS，这里通过 PRAGMA 检查再 ALTER。
	if err := ensureColumn(db, "datasets", "inference_mode", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "eval_tasks", "task_name", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	return nil
}

func ensureColumn(db *sql.DB, table, column, columnDef string) error {
	rows, err := db.QueryContext(context.Background(), fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("读取 %s 表结构失败: %w", table, err)
	}
	defer rows.Close()

	rowCount := 0
	for rows.Next() {
		rowCount++
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("扫描 %s 表结构失败: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历 %s 表结构失败: %w", table, err)
	}
	// PRAGMA 0 行 = 表不存在；fresh DB 下 001_init.sql 会负责建表，这里直接跳过避免 ALTER 不存在的表。
	if rowCount == 0 {
		return nil
	}

	if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnDef)); err != nil {
		return fmt.Errorf("ALTER TABLE %s 加列 %s 失败: %w", table, column, err)
	}
	log.Printf("数据库迁移：已为 %s 表添加 %s 列", table, column)
	return nil
}
