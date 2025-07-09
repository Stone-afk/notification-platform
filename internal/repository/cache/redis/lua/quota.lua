local key = KEYS[1]           -- 获取键名参数
local x = tonumber(ARGV[1])   -- 将传入参数x转换为数字
local current = tonumber(redis.call('GET', key) or 0) -- 处理键不存在的情况

if current < 0 then
    -- 负数时重置为x
    redis.call('SET', key, x)
    return x
elseif current > 0 then
    -- 正数时增加x（原子操作）
    return redis.call('INCRBY', key, x)
else
    -- 键不存在或值为0时设置为x
    redis.call('SET', key, x)
    return x
end