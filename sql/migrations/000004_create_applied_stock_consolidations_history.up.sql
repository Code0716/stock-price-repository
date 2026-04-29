CREATE TABLE applied_stock_consolidations_history (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    symbol VARCHAR(5) NOT NULL COMMENT '銘柄コード',
    consolidation_date DATE NOT NULL COMMENT '併合実施日',
    ratio DECIMAL(10, 4) NOT NULL COMMENT '併合比率（旧株数/新株数。例: 5株を1株に併合なら 5.0000）',
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '適用日時',
    UNIQUE KEY uq_symbol_consolidation_date (symbol, consolidation_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 検索効率向上のためのインデックス
CREATE INDEX idx_applied_stock_consolidations_history_symbol_date ON applied_stock_consolidations_history(symbol, consolidation_date);
