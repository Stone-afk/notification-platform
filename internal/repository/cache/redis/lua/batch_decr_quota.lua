-- 遍历所有键值对进行阈值检查
for i = 1, #KEYS do
    local key = KEYS[i]
    local threshold = tonumber(ARGV[i])
    local current = tonumber(redis.call('GET', key) or 0)

    -- 值不足时立即返回失败
    if current < threshold then
        return key
    end
end

-- 全部校验通过后执行扣减
for i = 1, #KEYS do
    local key = KEYS[i]
    local delta = tonumber(ARGV[i])
    redis.call('DECRBY', key, delta)
end

return ""