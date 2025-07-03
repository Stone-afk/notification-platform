CREATE TABLE `callback_log` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '回调记录ID',
    `notification_id` bigint unsigned NOT NULL COMMENT '待回调通知ID',
    `retry_count` tinyint NOT NULL DEFAULT '0' COMMENT '重试次数',
    `next_retry_time` bigint NOT NULL DEFAULT '0' COMMENT '下一次重试的时间戳',
    `status` enum('INIT','PENDING','SUCCEEDED','FAILED') NOT NULL DEFAULT 'INIT' COMMENT '回调状态',
    `ctime` bigint DEFAULT NULL,
    `utime` bigint DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_notification_id` (`notification_id`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='回调日志表';
