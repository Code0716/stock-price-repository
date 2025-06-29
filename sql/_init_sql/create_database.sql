DROP DATABASE IF EXISTS `stock_price_repository`;
CREATE DATABASE IF NOT EXISTS `stock_price_repository`;

-- stock_brand 銘柄
CREATE TABLE IF NOT EXISTS `stock_price_repository`.`stock_brand` (
  `id` CHAR(36) NOT NULL DEFAULT (LOWER(UUID())) COMMENT 'uuid',
  `ticker_symbol` VARCHAR(5) NOT NULL COMMENT '証券コード',
  `name` VARCHAR(255) NOT NULL COMMENT '銘柄名',
  `market_code` VARCHAR(255) NOT NULL COMMENT '市場コード',
  `market_name` VARCHAR(255) NOT NULL COMMENT '市場名',
  `sector_33_code` VARCHAR(4) DEFAULT NULL COMMENT '33業種コード',
  `sector_33_code_name` VARCHAR(255) DEFAULT NULL COMMENT '33業種区分',
  `sector_17_code` VARCHAR(4) DEFAULT NULL COMMENT '17業種コード',
  `sector_17_code_name` VARCHAR(255) DEFAULT NULL COMMENT '17業種区分',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'created_at',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'updated_at',
  `deleted_at` DATETIME DEFAULT NULL COMMENT 'deleted_at',
  PRIMARY KEY (`id`),
  INDEX idx_stock_brand_ticker_symbol (`ticker_symbol`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- nikkei_stock_average_daily 日経平均 日足
CREATE TABLE IF NOT EXISTS `stock_price_repository`.`nikkei_stock_average_daily_price` (
  `date` DATETIME NOT NULL COMMENT 'date',
  `open_price` DECIMAL(10, 4) NOT NULL COMMENT '始値',
  `close_price` DECIMAL(10, 4) NOT NULL COMMENT '終値',
  `high_price` DECIMAL(10, 4) NOT NULL COMMENT '高値',
  `low_price` DECIMAL(10, 4) NOT NULL COMMENT '安値',
  `adj_close_price` DECIMAL(10, 4) NOT NULL COMMENT '配当や株式分割を考慮した終値',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'created_at',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'updated_at',
  PRIMARY KEY (`date`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- dji_stock_average_daily NYダウ 日足
CREATE TABLE IF NOT EXISTS `stock_price_repository`.`dji_stock_average_daily_stock_price` (
  `date` DATETIME NOT NULL COMMENT 'date',
  `open_price` DECIMAL(10, 4) NOT NULL COMMENT '始値',
  `close_price` DECIMAL(10, 4) NOT NULL COMMENT '終値',
  `high_price` DECIMAL(10, 4) NOT NULL COMMENT '高値',
  `low_price` DECIMAL(10, 4) NOT NULL COMMENT '安値',
  `adj_close_price` DECIMAL(10, 4) NOT NULL COMMENT '配当や株式分割を考慮した終値',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'created_at',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'updated_at',
  PRIMARY KEY (`date`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- stock_brands_daily_price 個別銘柄 - 日足
CREATE TABLE IF NOT EXISTS `stock_price_repository`.`stock_brands_daily_price` (
  `id` CHAR(36) NOT NULL COMMENT 'uuid',
  `stock_brand_id` CHAR(36) NOT NULL COMMENT 'uuid',
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
  `deleted_at` DATETIME DEFAULT NULL COMMENT 'deleted_at',
  PRIMARY KEY (`id`),
  INDEX idx_stock_brands_daily_stock_price_ticker_symbol (`ticker_symbol`),
  INDEX idx_stock_brands_daily_stock_price_date (`date`),
  INDEX idx_stock_brands_daily_stock_price_stock_brand_id_and_date (`stock_brand_id`, `date`),
  -- stock_brand_idとdateが一緒のものは入れない
  UNIQUE KEY unique_stock_brands_daily_stock_price_stock_brand_id_and_date (`stock_brand_id`, `date`),
  FOREIGN KEY (`stock_brand_id`) REFERENCES stock_brand (`id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;


--  33業種平均価格 - 日足
CREATE TABLE IF NOT EXISTS `stock_price_repository`.`sector_33_average_daily_price` (
  `id` CHAR(36) NOT NULL COMMENT 'uuid',
  `date` DATE NOT NULL COMMENT 'date',
  `sector_33_code` VARCHAR(4) DEFAULT NULL COMMENT '33業種コード',
  `open_price` DECIMAL(10, 4) NOT NULL COMMENT '始値',
  `close_price` DECIMAL(10, 4) NOT NULL COMMENT '終値',
  `high_price` DECIMAL(10, 4) NOT NULL COMMENT '高値',
  `low_price` DECIMAL(10, 4) NOT NULL COMMENT '安値',
  `adj_close_price` DECIMAL(10, 4) NOT NULL COMMENT '配当や株式分割を考慮した終値',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'created_at',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'updated_at',
  PRIMARY KEY (`id`),
  UNIQUE KEY idx_sector_33_average_date_and_code (`date`, `sector_33_code`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

--  17業種平均価格 - 日足
CREATE TABLE IF NOT EXISTS `stock_price_repository`.`sector_17_average_daily_price` (
  `id` CHAR(36) NOT NULL COMMENT 'uuid',
  `date` DATE NOT NULL COMMENT 'date',
  `sector_33_code` VARCHAR(4) DEFAULT NULL COMMENT '33業種コード',
  `sector_17_code` VARCHAR(4) DEFAULT NULL COMMENT '17業種コード',
  `open_price` DECIMAL(10, 4) NOT NULL COMMENT '始値',
  `close_price` DECIMAL(10, 4) NOT NULL COMMENT '終値',
  `high_price` DECIMAL(10, 4) NOT NULL COMMENT '高値',
  `low_price` DECIMAL(10, 4) NOT NULL COMMENT '安値',
  `adj_close_price` DECIMAL(10, 4) NOT NULL COMMENT '配当や株式分割を考慮した終値',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'created_at',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'updated_at',
  PRIMARY KEY (`id`),
  UNIQUE KEY idx_sector_17_average_date_and_sector_code (`date`, `sector_17_code`),
  UNIQUE KEY idx_sector_17_average_date_and_17_and_33_code (`date`, `sector_17_code`, `sector_33_code`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

