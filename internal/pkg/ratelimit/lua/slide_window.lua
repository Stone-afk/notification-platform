-- 滑动窗口限流算法

-- 限流对象的请求计数键
local countKey = KEYS[1]
-- 限流对象的限流事件记录键
local limitedEventKey = KEYS[2]
-- 窗口大小（毫秒）
local window = tonumber(ARGV[1])
-- 阈值（最大请求数）
local threshold = tonumber(ARGV[2])
-- 当前时间戳（毫秒）
local now = tonumber(ARGV[3])
-- 窗口的起始时间
local min = now - window

-- 移除窗口外的所有记录
redis.call('ZREMRANGEBYSCORE', countKey, '-inf', min)
-- 计算窗口内的请求数
local cnt = redis.call('ZCOUNT', countKey, min, '+inf')

if cnt >= threshold then
    -- 执行限流并只记录最新的限流时间
    redis.call('SET', limitedEventKey, now, 'PX', 86400000)  -- 保留一天
    return "true"
else
    -- 记录当前请求，使用当前时间戳作为score和member
    redis.call('ZADD', countKey, now, now)
    -- 设置过期时间，避免长期占用内存
    redis.call('PEXPIRE', countKey, window)
    return "false"
end  