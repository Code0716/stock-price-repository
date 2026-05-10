ALTER TABLE `analyze_stock_brand_price_history`
  DROP INDEX `uq_analyze_stock_brand_price_history_stock_ticker_method_date`;

ALTER TABLE `analyze_stock_brand_price_history`
  ADD CONSTRAINT `uq_analyze_stock_brand_price_history_stock_ticker_method`
    UNIQUE (`stock_brand_id`, `ticker_symbol`, `method`);

ALTER TABLE `analyze_stock_brand_price_history`
  DROP INDEX `idx_analyze_stock_brand_price_history_stock_brand_id`;
