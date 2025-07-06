

CREATE TABLE quota (
   id BIGINT UNSIGNED PRIMARY KEY COMMENT '雪花算法ID',
   biz_id BIGINT NOT NULL COMMENT '业务配表ID，业务方可能有多个业务每个业务配置不同',
   channel ENUM('SMS','EMAIL','IN_APP') NOT NULL COMMENT '发送渠道',
   quota INT COMMENT '每个月的 quota',
   utime BIGINT COMMENT '时间戳，毫秒数',
   ctime BIGINT COMMENT '时间戳，毫秒数',
   UNIQUE INDEX biz_id_channel (biz_id, channel)
) COMMENT='Quota table';