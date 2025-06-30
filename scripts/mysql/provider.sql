
CREATE TABLE `provider` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '供应商ID',
    `name` VARCHAR(64) NOT NULL COMMENT '供应商名称',
    `channel` ENUM('SMS','EMAIL','IN_APP') NOT NULL COMMENT '支持的渠道',
    `endpoint` VARCHAR(255) NOT NULL COMMENT 'API入口地址',
    `region_id` VARCHAR(255) COMMENT '区域ID',
    `api_key` VARCHAR(255) NOT NULL COMMENT 'API密钥，明文',
    `api_secret` VARCHAR(512) NOT NULL COMMENT 'API密钥,加密',
    `appid` VARCHAR(512) COMMENT '应用ID，仅腾讯云使用',
    `weight` INT NOT NULL COMMENT '权重',
    `qps_limit` INT NOT NULL COMMENT '每秒请求数限制',
    `daily_limit` INT NOT NULL COMMENT '每日请求数限制',
    `audit_callback_url` VARCHAR(256) COMMENT '回调URL，供应商通知审核结果',
    `status` ENUM('ACTIVE','INACTIVE') NOT NULL DEFAULT 'ACTIVE' COMMENT '状态，启用-ACTIVE，禁用-INACTIVE',
    `ctime` BIGINT,
    `utime` BIGINT,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_name_channel` (`name`, `channel`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='供应商信息表';