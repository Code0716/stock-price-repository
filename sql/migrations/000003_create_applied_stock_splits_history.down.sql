-- インデックスの削除
DROP INDEX idx_applied_stock_splits_history_symbol_split_date ON applied_stock_splits_history;

DROP TABLE IF EXISTS applied_stock_splits_history;
