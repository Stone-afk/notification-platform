
CREATE TABLE `channel_template` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '渠道模版ID',
    `owner_id` BIGINT NOT NULL COMMENT '用户ID或部门ID',
    `owner_type` ENUM('person', 'organization') NOT NULL COMMENT '业务方类型：person-个人,organization-组织',
    `name` VARCHAR(128) NOT NULL COMMENT '模板名称',
    `description` VARCHAR(512) NOT NULL COMMENT '模板描述',
    `channel` ENUM('SMS','EMAIL','IN_APP') NOT NULL COMMENT '渠道类型',
    `business_type` BIGINT NOT NULL DEFAULT 1 COMMENT '业务类型：1-推广营销、2-通知、3-验证码等',
    `active_version_id` BIGINT DEFAULT 0 COMMENT '当前启用的版本ID，0表示无活跃版本',
    `ctime` BIGINT,
    `utime` BIGINT,
    PRIMARY KEY (`id`),
    KEY `idx_active_version` (`active_version_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='渠道模板表';

CREATE TABLE `channel_template_version` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '渠道模版版本ID',
    `channel_template_id` BIGINT NOT NULL COMMENT '关联渠道模版ID',
    `name` VARCHAR(32) NOT NULL COMMENT '版本名称，如v1.0.0',
    `signature` VARCHAR(64) COMMENT '已通过所有供应商审核的短信签名/邮件发件人',
    `content` TEXT NOT NULL COMMENT '原始模板内容，使用平台统一变量格式，如${name}',
    `remark` TEXT NOT NULL COMMENT '申请说明,描述使用短信的业务场景，并提供短信完整示例（填入变量内容），信息完整有助于提高模板审核通过率。',
    `audit_id` BIGINT NOT NULL DEFAULT 0 COMMENT '审核表ID, 0表示尚未提交审核或者未拿到审核结果',
    `auditor_id` BIGINT COMMENT '审核人ID',
    `audit_time` BIGINT COMMENT '审核时间',
    `audit_status` ENUM('PENDING','IN_REVIEW','REJECTED','APPROVED') NOT NULL DEFAULT 'PENDING' COMMENT '内部审核状态',
    `reject_reason` VARCHAR(512) COMMENT '拒绝原因',
    `last_review_submission_time` BIGINT COMMENT '上一次提交审核时间',
    `ctime` BIGINT,
    `utime` BIGINT,
    PRIMARY KEY (`id`),
    KEY `idx_channel_template_id` (`channel_template_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='渠道模板版本表';

CREATE TABLE `channel_template_provider` (
     `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '渠道模版-供应商关联ID',
     `template_id` BIGINT NOT NULL COMMENT '渠道模版ID',
     `template_version_id` BIGINT NOT NULL COMMENT '渠道模版版本ID',
     `provider_id` BIGINT NOT NULL COMMENT '供应商ID',
     `provider_name` VARCHAR(64) NOT NULL COMMENT '供应商名称',
     `provider_channel` ENUM('SMS','EMAIL','IN_APP') NOT NULL COMMENT '渠道类型',
     `request_id` VARCHAR(256) COMMENT '审核请求在供应商侧的ID，用于排查问题',
     `provider_template_id` VARCHAR(256) COMMENT '当前版本模版在供应商侧的ID，审核通过后才会有值',
     `audit_status` ENUM('PENDING','IN_REVIEW','REJECTED','APPROVED') NOT NULL DEFAULT 'PENDING' COMMENT '供应商侧模版审核状态',
     `reject_reason` VARCHAR(512) COMMENT '供应商侧拒绝原因',
     `last_review_submission_time` BIGINT COMMENT '上一次提交审核时间',
     `ctime` BIGINT,
     `utime` BIGINT,
     PRIMARY KEY (`id`),
     UNIQUE KEY `idx_template_version_provider` (`template_id`, `template_version_id`, `provider_id`),
     UNIQUE KEY `idx_tmpl_ver_name_chan` (`template_id`, `template_version_id`, `provider_name`, `provider_channel`),
     KEY `idx_request_id` (`request_id`),
     KEY `idx_audit_status` (`audit_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='渠道模板-供应商关联表';