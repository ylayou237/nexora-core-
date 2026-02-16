-- KEYS[1] : session_key (ex: session:acct-123)
-- ARGV[1] : input_increment (bytes)
-- ARGV[2] : output_increment (bytes)
-- ARGV[3] : now_timestamp (unix)

local key = KEYS[1]
local in_inc = tonumber(ARGV[1])
local out_inc = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- 1. Récupération atomique des compteurs et limites
local data = redis.call("HMGET", key, "used_in", "used_out", "quota", "expires_at")
local used_in = tonumber(data[1] or 0)
local used_out = tonumber(data[2] or 0)
local quota = tonumber(data[3] or 0)
local expires_at = tonumber(data[4] or 0)

-- 2. Vérification Temps (Lease expiration)
if expires_at > 0 and now > expires_at then
    return 0 -- Signal : Expired
end

-- 3. Vérification Quota (Download + Upload)
local total_new = used_in + used_out + in_inc + out_inc
if quota > 0 and total_new > quota then
    return -1 -- Signal : Quota Exceeded
end

-- 4. Mise à jour atomique si tout est OK
redis.call("HINCRBY", key, "used_in", in_inc)
redis.call("HINCRBY", key, "used_out", out_inc)
redis.call("HSET", key, "last_update", now)

return 1 -- Signal : Success