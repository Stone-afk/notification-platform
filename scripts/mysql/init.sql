-- create the databases
CREATE
DATABASE IF NOT EXISTS `notification`;

-- create the users for each database
CREATE
USER 'notification'@'%' IDENTIFIED BY 'notification';
GRANT CREATE
, ALTER
, INDEX, LOCK TABLES, REFERENCES,
UPDATE,
DELETE
, DROP
,
SELECT,
INSERT
ON `notification`.* TO 'notification'@'%';

FLUSH
PRIVILEGES;


CREATE
DATABASE IF NOT EXISTS `notification_0`;

use notification_0;

CREATE TABLE `callback_log_0`
(
    `id`              BIGINT  NOT NULL AUTO_INCREMENT COMMENT '回调记录ID',
    `notification_id` BIGINT UNSIGNED NOT NULL COMMENT '待回调通知ID',
    `retry_count`     TINYINT NOT NULL DEFAULT 0 COMMENT '重试次数',
    `next_retry_time` BIGINT  NOT NULL DEFAULT 0 COMMENT '下一次重试的时间戳',
    `status`          ENUM('INIT','PENDING','SUCCEEDED','FAILED') NOT NULL DEFAULT 'INIT' COMMENT '回调状态',
    `ctime`           BIGINT  NOT NULL,
    `utime`           BIGINT  NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_notification_id` (`notification_id`),
    INDEX             `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='回调记录表';

CREATE TABLE `callback_log_1`
(
    `id`              BIGINT  NOT NULL AUTO_INCREMENT COMMENT '回调记录ID',
    `notification_id` BIGINT UNSIGNED NOT NULL COMMENT '待回调通知ID',
    `retry_count`     TINYINT NOT NULL DEFAULT 0 COMMENT '重试次数',
    `next_retry_time` BIGINT  NOT NULL DEFAULT 0 COMMENT '下一次重试的时间戳',
    `status`          ENUM('INIT','PENDING','SUCCEEDED','FAILED') NOT NULL DEFAULT 'INIT' COMMENT '回调状态',
    `ctime`           BIGINT  NOT NULL,
    `utime`           BIGINT  NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_notification_id` (`notification_id`),
    INDEX             `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='回调记录表';

CREATE TABLE `notification_0`
(
    `id`                  BIGINT UNSIGNED NOT NULL COMMENT '雪花算法ID',
    `biz_id`              BIGINT       NOT NULL COMMENT '业务配表ID，业务方可能有多个业务每个业务配置不同',
    `key`                 VARCHAR(256) NOT NULL COMMENT '业务内唯一标识，区分同一个业务内的不同通知',
    `receivers`           TEXT         NOT NULL COMMENT '接收者(手机/邮箱/用户ID)，JSON数组',
    `channel`             ENUM('SMS','EMAIL','IN_APP') NOT NULL COMMENT '发送渠道',
    `template_id`         BIGINT       NOT NULL COMMENT '模板ID',
    `template_version_id` BIGINT       NOT NULL COMMENT '模板版本ID',
    `template_params`     TEXT         NOT NULL COMMENT '模版参数',
    `status`              ENUM('PREPARE','CANCELED','PENDING','SENDING','SUCCEEDED','FAILED') DEFAULT 'PENDING' COMMENT '发送状态',
    `scheduled_stime`     BIGINT       NOT NULL COMMENT '计划发送开始时间',
    `scheduled_etime`     BIGINT       NOT NULL COMMENT '计划发送结束时间',
    `version`             INT          NOT NULL DEFAULT 1 COMMENT '版本号，用于CAS操作',
    `ctime`               BIGINT       NOT NULL,
    `utime`               BIGINT       NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_biz_id_key` (`biz_id`, `key`),
    INDEX                 `idx_biz_id_status` (`biz_id`, `status`),
    INDEX                 `idx_scheduled` (`scheduled_stime`, `scheduled_etime`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知记录表';

CREATE TABLE `notification_1`
(
    `id`                  BIGINT UNSIGNED NOT NULL COMMENT '雪花算法ID',
    `biz_id`              BIGINT       NOT NULL COMMENT '业务配表ID，业务方可能有多个业务每个业务配置不同',
    `key`                 VARCHAR(256) NOT NULL COMMENT '业务内唯一标识，区分同一个业务内的不同通知',
    `receivers`           TEXT         NOT NULL COMMENT '接收者(手机/邮箱/用户ID)，JSON数组',
    `channel`             ENUM('SMS','EMAIL','IN_APP') NOT NULL COMMENT '发送渠道',
    `template_id`         BIGINT       NOT NULL COMMENT '模板ID',
    `template_version_id` BIGINT       NOT NULL COMMENT '模板版本ID',
    `template_params`     TEXT         NOT NULL COMMENT '模版参数',
    `status`              ENUM('PREPARE','CANCELED','PENDING','SENDING','SUCCEEDED','FAILED') DEFAULT 'PENDING' COMMENT '发送状态',
    `scheduled_stime`     BIGINT       NOT NULL COMMENT '计划发送开始时间',
    `scheduled_etime`     BIGINT       NOT NULL COMMENT '计划发送结束时间',
    `version`             INT          NOT NULL DEFAULT 1 COMMENT '版本号，用于CAS操作',
    `ctime`               BIGINT       NOT NULL,
    `utime`               BIGINT       NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_biz_id_key` (`biz_id`, `key`),
    INDEX                 `idx_biz_id_status` (`biz_id`, `status`),
    INDEX                 `idx_scheduled` (`scheduled_stime`, `scheduled_etime`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知记录表';

CREATE TABLE `tx_notification_0`
(
    `tx_id`           BIGINT       NOT NULL AUTO_INCREMENT COMMENT '事务ID',
    `key`             VARCHAR(256) NOT NULL COMMENT '业务内唯一标识，区分同一个业务内的不同通知',
    `notification_id` BIGINT UNSIGNED NOT NULL COMMENT '创建的通知ID',
    `biz_id`          BIGINT       NOT NULL COMMENT '业务方唯一标识',
    `status`          VARCHAR(20)  NOT NULL DEFAULT 'PREPARE' COMMENT '通知状态',
    `check_count`     INT          NOT NULL DEFAULT 1 COMMENT '第几次检查从1开始',
    `next_check_time` BIGINT       NOT NULL DEFAULT 0 COMMENT '下一次的回查时间戳',
    `ctime`           BIGINT       NOT NULL COMMENT '创建时间',
    `utime`           BIGINT       NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`tx_id`),
    UNIQUE INDEX `idx_biz_id_key` (`biz_id`, `key`),
    INDEX             `idx_next_check_time_status` (`next_check_time`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事务通知表';

CREATE TABLE `tx_notification_1`
(
    `tx_id`           BIGINT       NOT NULL AUTO_INCREMENT COMMENT '事务ID',
    `key`             VARCHAR(256) NOT NULL COMMENT '业务内唯一标识，区分同一个业务内的不同通知',
    `notification_id` BIGINT UNSIGNED NOT NULL COMMENT '创建的通知ID',
    `biz_id`          BIGINT       NOT NULL COMMENT '业务方唯一标识',
    `status`          VARCHAR(20)  NOT NULL DEFAULT 'PREPARE' COMMENT '通知状态',
    `check_count`     INT          NOT NULL DEFAULT 1 COMMENT '第几次检查从1开始',
    `next_check_time` BIGINT       NOT NULL DEFAULT 0 COMMENT '下一次的回查时间戳',
    `ctime`           BIGINT       NOT NULL COMMENT '创建时间',
    `utime`           BIGINT       NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`tx_id`),
    UNIQUE INDEX `idx_biz_id_key` (`biz_id`, `key`),
    INDEX             `idx_next_check_time_status` (`next_check_time`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事务通知表';

CREATE
DATABASE IF NOT EXISTS `notification_1`;

use notification_1;

CREATE TABLE `callback_log_0`
(
    `id`              BIGINT  NOT NULL AUTO_INCREMENT COMMENT '回调记录ID',
    `notification_id` BIGINT UNSIGNED NOT NULL COMMENT '待回调通知ID',
    `retry_count`     TINYINT NOT NULL DEFAULT 0 COMMENT '重试次数',
    `next_retry_time` BIGINT  NOT NULL DEFAULT 0 COMMENT '下一次重试的时间戳',
    `status`          ENUM('INIT','PENDING','SUCCEEDED','FAILED') NOT NULL DEFAULT 'INIT' COMMENT '回调状态',
    `ctime`           BIGINT  NOT NULL,
    `utime`           BIGINT  NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_notification_id` (`notification_id`),
    INDEX             `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='回调记录表';

CREATE TABLE `callback_log_1`
(
    `id`              BIGINT  NOT NULL AUTO_INCREMENT COMMENT '回调记录ID',
    `notification_id` BIGINT UNSIGNED NOT NULL COMMENT '待回调通知ID',
    `retry_count`     TINYINT NOT NULL DEFAULT 0 COMMENT '重试次数',
    `next_retry_time` BIGINT  NOT NULL DEFAULT 0 COMMENT '下一次重试的时间戳',
    `status`          ENUM('INIT','PENDING','SUCCEEDED','FAILED') NOT NULL DEFAULT 'INIT' COMMENT '回调状态',
    `ctime`           BIGINT  NOT NULL,
    `utime`           BIGINT  NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_notification_id` (`notification_id`),
    INDEX             `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='回调记录表';

CREATE TABLE `notification_0`
(
    `id`                  BIGINT UNSIGNED NOT NULL COMMENT '雪花算法ID',
    `biz_id`              BIGINT       NOT NULL COMMENT '业务配表ID，业务方可能有多个业务每个业务配置不同',
    `key`                 VARCHAR(256) NOT NULL COMMENT '业务内唯一标识，区分同一个业务内的不同通知',
    `receivers`           TEXT         NOT NULL COMMENT '接收者(手机/邮箱/用户ID)，JSON数组',
    `channel`             ENUM('SMS','EMAIL','IN_APP') NOT NULL COMMENT '发送渠道',
    `template_id`         BIGINT       NOT NULL COMMENT '模板ID',
    `template_version_id` BIGINT       NOT NULL COMMENT '模板版本ID',
    `template_params`     TEXT         NOT NULL COMMENT '模版参数',
    `status`              ENUM('PREPARE','CANCELED','PENDING','SENDING','SUCCEEDED','FAILED') DEFAULT 'PENDING' COMMENT '发送状态',
    `scheduled_stime`     BIGINT       NOT NULL COMMENT '计划发送开始时间',
    `scheduled_etime`     BIGINT       NOT NULL COMMENT '计划发送结束时间',
    `version`             INT          NOT NULL DEFAULT 1 COMMENT '版本号，用于CAS操作',
    `ctime`               BIGINT       NOT NULL,
    `utime`               BIGINT       NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_biz_id_key` (`biz_id`, `key`),
    INDEX                 `idx_biz_id_status` (`biz_id`, `status`),
    INDEX                 `idx_scheduled` (`scheduled_stime`, `scheduled_etime`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知记录表';

CREATE TABLE `notification_1`
(
    `id`                  BIGINT UNSIGNED NOT NULL COMMENT '雪花算法ID',
    `biz_id`              BIGINT       NOT NULL COMMENT '业务配表ID，业务方可能有多个业务每个业务配置不同',
    `key`                 VARCHAR(256) NOT NULL COMMENT '业务内唯一标识，区分同一个业务内的不同通知',
    `receivers`           TEXT         NOT NULL COMMENT '接收者(手机/邮箱/用户ID)，JSON数组',
    `channel`             ENUM('SMS','EMAIL','IN_APP') NOT NULL COMMENT '发送渠道',
    `template_id`         BIGINT       NOT NULL COMMENT '模板ID',
    `template_version_id` BIGINT       NOT NULL COMMENT '模板版本ID',
    `template_params`     TEXT         NOT NULL COMMENT '模版参数',
    `status`              ENUM('PREPARE','CANCELED','PENDING','SENDING','SUCCEEDED','FAILED') DEFAULT 'PENDING' COMMENT '发送状态',
    `scheduled_stime`     BIGINT       NOT NULL COMMENT '计划发送开始时间',
    `scheduled_etime`     BIGINT       NOT NULL COMMENT '计划发送结束时间',
    `version`             INT          NOT NULL DEFAULT 1 COMMENT '版本号，用于CAS操作',
    `ctime`               BIGINT       NOT NULL,
    `utime`               BIGINT       NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_biz_id_key` (`biz_id`, `key`),
    INDEX                 `idx_biz_id_status` (`biz_id`, `status`),
    INDEX                 `idx_scheduled` (`scheduled_stime`, `scheduled_etime`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知记录表';


CREATE TABLE `tx_notification_0`
(
    `tx_id`           BIGINT       NOT NULL AUTO_INCREMENT COMMENT '事务ID',
    `key`             VARCHAR(256) NOT NULL COMMENT '业务内唯一标识，区分同一个业务内的不同通知',
    `notification_id` BIGINT UNSIGNED NOT NULL COMMENT '创建的通知ID',
    `biz_id`          BIGINT       NOT NULL COMMENT '业务方唯一标识',
    `status`          VARCHAR(20)  NOT NULL DEFAULT 'PREPARE' COMMENT '通知状态',
    `check_count`     INT          NOT NULL DEFAULT 1 COMMENT '第几次检查从1开始',
    `next_check_time` BIGINT       NOT NULL DEFAULT 0 COMMENT '下一次的回查时间戳',
    `ctime`           BIGINT       NOT NULL COMMENT '创建时间',
    `utime`           BIGINT       NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`tx_id`),
    UNIQUE INDEX `idx_biz_id_key` (`biz_id`, `key`),
    INDEX             `idx_next_check_time_status` (`next_check_time`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事务通知表';

CREATE TABLE `tx_notification_1`
(
    `tx_id`           BIGINT       NOT NULL AUTO_INCREMENT COMMENT '事务ID',
    `key`             VARCHAR(256) NOT NULL COMMENT '业务内唯一标识，区分同一个业务内的不同通知',
    `notification_id` BIGINT UNSIGNED NOT NULL COMMENT '创建的通知ID',
    `biz_id`          BIGINT       NOT NULL COMMENT '业务方唯一标识',
    `status`          VARCHAR(20)  NOT NULL DEFAULT 'PREPARE' COMMENT '通知状态',
    `check_count`     INT          NOT NULL DEFAULT 1 COMMENT '第几次检查从1开始',
    `next_check_time` BIGINT       NOT NULL DEFAULT 0 COMMENT '下一次的回查时间戳',
    `ctime`           BIGINT       NOT NULL COMMENT '创建时间',
    `utime`           BIGINT       NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`tx_id`),
    UNIQUE INDEX `idx_biz_id_key` (`biz_id`, `key`),
    INDEX             `idx_next_check_time_status` (`next_check_time`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事务通知表';