CREATE TABLE applied_stock_splits_history (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    symbol VARCHAR(5) NOT NULL COMMENT '銘柄コード',
    split_date DATE NOT NULL COMMENT '分割実施日',
    ratio DECIMAL(10, 4) NOT NULL COMMENT '分割比率',
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '適用日時',
    UNIQUE KEY uq_symbol_split_date (symbol, split_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 検索効率向上のためのインデックス
CREATE INDEX idx_applied_stock_splits_history_symbol_split_date ON applied_stock_splits_history(symbol, split_date);
