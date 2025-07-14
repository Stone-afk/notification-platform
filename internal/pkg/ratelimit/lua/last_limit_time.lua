-- 获取最后一次限流事件的时间

-- 限流对象的限流事件记录键
local limitedEventKey = KEYS[1]

-- 直接获取时间戳
local timestamp = redis.call('GET', limitedEventKey)

if timestamp then
    return timestamp
else
    return "0"
end