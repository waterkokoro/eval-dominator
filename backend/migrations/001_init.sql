CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- models：模型预设。每条记录是一份 OpenAI 兼容接口（provider + 模型名 + base_url + api_key）。
CREATE TABLE IF NOT EXISTS models (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    provider TEXT NOT NULL,
    model_name TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    base_url TEXT,
    api_key TEXT NOT NULL,
    masked_key TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS eval_tasks (
    id TEXT PRIMARY KEY,
    task_name TEXT NOT NULL DEFAULT '',
    user_id INTEGER NOT NULL,
    model_provider TEXT NOT NULL,
    model_name TEXT NOT NULL,
    model_base_url TEXT,
    dataset_type TEXT NOT NULL,
    dataset_name TEXT NOT NULL,
    status TEXT NOT NULL,
    output_dir TEXT,
    error_code TEXT,
    error_message TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    finished_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS eval_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    eval_task_id TEXT NOT NULL UNIQUE,
    metrics_json TEXT NOT NULL,
    artifacts_json TEXT NOT NULL,
    raw_result_path TEXT,
    report_path TEXT,
    log_path TEXT,
    metadata_json TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (eval_task_id) REFERENCES eval_tasks(id)
);

CREATE TABLE IF NOT EXISTS datasets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'opencompass_demo',
    source TEXT NOT NULL DEFAULT 'builtin',
    sample_count INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    inference_mode TEXT NOT NULL DEFAULT '',
    config_path TEXT NOT NULL DEFAULT '',
    extra_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 升级到老版本数据库时需要补的列由 Go 端 ensureColumn() 处理（SQLite 不支持
-- ADD COLUMN IF NOT EXISTS）。fresh DB 不会用到。
