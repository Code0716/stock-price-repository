-- 既存ユニークインデックスを削除
ALTER TABLE `analyze_stock_brand_price_history`
  DROP INDEX `unique_symbol_action_method_created_at`;

-- 複合ユニークキーを追加
ALTER TABLE `analyze_stock_brand_price_history`
  ADD CONSTRAINT `uq_analyze_stock_brand_price_history_stock_ticker_method`
    UNIQUE (`stock_brand_id`, `ticker_symbol`, `method`);
