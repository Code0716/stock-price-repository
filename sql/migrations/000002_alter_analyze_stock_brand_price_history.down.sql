-- 複合ユニーク（作成時に CONSTRAINT 名を付けた場合）
ALTER TABLE `analyze_stock_brand_price_history`
  DROP CONSTRAINT `uq_analyze_stock_brand_price_history_stock_ticker_method`;

-- 元のユニークキーを復元
ALTER TABLE `analyze_stock_brand_price_history`
  ADD UNIQUE KEY `unique_symbol_action_method_created_at`
    (`ticker_symbol`, `action`, `method`, `created_at`);
