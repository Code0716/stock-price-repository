ALTER TABLE `analyze_stock_brand_price_history`
  ADD COLUMN `current_price` DECIMAL(10, 4) NOT NULL DEFAULT 0 COMMENT '現在値' AFTER `trade_price`;
