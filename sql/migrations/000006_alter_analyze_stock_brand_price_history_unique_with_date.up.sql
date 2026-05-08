-- FKが既存ユニークキーのインデックスに依存しているため、先に補助インデックスを追加する
ALTER TABLE `analyze_stock_brand_price_history`
  ADD INDEX `idx_analyze_stock_brand_price_history_stock_brand_id` (`stock_brand_id`);

ALTER TABLE `analyze_stock_brand_price_history`
  DROP INDEX `uq_analyze_stock_brand_price_history_stock_ticker_method`;

ALTER TABLE `analyze_stock_brand_price_history`
  ADD CONSTRAINT `uq_analyze_stock_brand_price_history_stock_ticker_method_date`
    UNIQUE (`stock_brand_id`, `ticker_symbol`, `method`, `created_at`);
