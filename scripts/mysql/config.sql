
CREATE TABLE `business_config` (
       `id` bigint NOT NULL COMMENT '业务标识',
       `owner_id` bigint DEFAULT NULL COMMENT '业务方',
       `owner_type` enum('person','organization') DEFAULT NULL COMMENT '业务方类型：person-个人,organization-组织',
       `channel_config` json DEFAULT NULL COMMENT '{"channels":[{"channel":"SMS", "priority":"1","enabled":"true"},{"channel":"EMAIL", "priority":"2","enabled":"true"}]}',
       `txn_config` json DEFAULT NULL COMMENT '事务配置',
       `rate_limit` int DEFAULT '1000' COMMENT '每秒最大请求数',
       `quota` json DEFAULT NULL COMMENT '{"monthly":{"SMS":100000,"EMAIL":500000}}',
       `callback_config` json DEFAULT NULL COMMENT '回调配置，通知平台回调业务方通知异步请求结果',
       `ctime` bigint DEFAULT NULL,
       `utime` bigint DEFAULT NULL,
       PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='业务配置表';