-- KEYS[1] : session_key (ex: session:acct-123)
-- ARGV[1] : input_inc (octets entrants)
-- ARGV[2] : output_inc (octets sortants)
-- ARGV[3] : now_timestamp (unix epoch)
-- ARGV[4] : grace_period (secondes, pour garder la clé après expiration pour audit)

local key = KEYS[1]
local in_inc = tonumber(ARGV[1]) or 0
local out_inc = tonumber(ARGV[2]) or 0
local now = tonumber(ARGV[3])
local grace = tonumber(ARGV[4]) or 3600

-- 1. Récupération atomique de l'état
local data = redis.call("HMGET", key, "used_in", "used_out", "quota", "expires_at", "status")
local used_in = tonumber(data[1] or 0)
local used_out = tonumber(data[2] or 0)
local quota = tonumber(data[3] or 0)
local expires_at = tonumber(data[4] or 0)
local status = data[5]

-- 2. Validation d'existence et de statut
if not status then
    return -2 -- Signal : SESSION_NOT_FOUND
end
if status == "suspended" or status == "revoked" then
    return -3 -- Signal : SESSION_LOCKED
end

-- 3. Vérification de la validité du bail (Lease)
if expires_at > 0 and now > expires_at then
    -- On marque le statut en base pour éviter de recalculer au prochain tour
    redis.call("HSET", key, "status", "expired")
    return 0 -- Signal : SESSION_EXPIRED
end

-- 4. Vérification de Quota (In + Out)
-- Carrier-grade : on calcule la projection AVANT d'incrémenter
local projected_total = used_in + used_out + in_inc + out_inc
if quota > 0 and projected_total > quota then
    -- On log l'événement de dépassement dans une liste Redis pour le SIEM
    local event = string.format('{"ts":%d, "msg":"quota_exceeded", "limit":%d, "attempted":%d}', now, quota, projected_total)
    redis.call("LPUSH", "events:quota:violation", event)
    redis.call("LTRIM", "events:quota:violation", 0, 999) -- On garde les 1000 derniers
    return -1 -- Signal : QUOTA_EXCEEDED
end

-- 5. Mise à jour atomique
-- HINCRBYFLOAT est utilisé pour supporter des volumes massifs si nécessaire (TeraBytes)
redis.call("HINCRBY", key, "used_in", in_inc)
redis.call("HINCRBY", key, "used_out", out_inc)
redis.call("HMSET", key, "last_update", now, "status", "active")

-- 6. Synchronisation du TTL Redis avec la logique métier
-- Indispensable pour ne pas fuiter de la mémoire dans Redis
if expires_at > 0 then
    local diff = expires_at - now
    redis.call("EXPIRE", key, diff + grace)
end

return 1 -- Signal : SUCCESS