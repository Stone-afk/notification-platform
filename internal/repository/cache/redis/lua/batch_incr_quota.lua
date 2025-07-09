-- 遍历所有键值对
for i = 1, #KEYS do
    local key = KEYS[i]
    local param = tonumber(ARGV[i])
    local current = tonumber(redis.call('GET', key) or 0)  -- 处理键不存在的情况

    if current < 0 then
        -- 值小于0时设置为参数值
        redis.call('SET', key, param)
    else
        -- 值≥0时增加参数值（原子操作）
        redis.call('INCRBY', key, param)
    end
end

return 1  -- 返回成功标志