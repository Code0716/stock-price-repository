-- 分析用の個別銘柄日足
CREATE TABLE IF NOT EXISTS `stock_price_repository`.`stock_brands_daily_price_for_analyze` (
  `id` char(36) NOT NULL COMMENT 'uuid',
  `ticker_symbol` VARCHAR(36) NOT NULL COMMENT 'ticker symbol',
  `date` DATE NOT NULL COMMENT 'date',
  `open_price` DECIMAL(10, 4) NOT NULL COMMENT '始値',
  `close_price` DECIMAL(10, 4) NOT NULL COMMENT '終値',
  `high_price` DECIMAL(10, 4) NOT NULL COMMENT '高値',
  `low_price` DECIMAL(10, 4) NOT NULL COMMENT '安値',
  `adj_close_price` DECIMAL(10, 4) NOT NULL COMMENT '配当や株式分割を考慮した終値',
  `volume` BIGINT UNSIGNED NOT NULL COMMENT '出来高',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'created_at',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'updated_at',
  PRIMARY KEY (`id`),
  INDEX idx_stock_brands_daily_price_ticker_symbol (`ticker_symbol`),
  INDEX idx_stock_brands_daily_price_date (`date`),
  INDEX idx_stock_brands_daily_price_ticker_symbol_and_date (`ticker_symbol`, `date`),
  -- シンボルとdateが一緒のものは入れない
  UNIQUE KEY unique_stock_brands_daily_price_ticker_symbol_and_date (`ticker_symbol`, `date`)
--   FOREIGN KEY (`ticker_symbol`) REFERENCES stock_brand (ticker_symbol)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;
