-- KEYS[1]: old_key (rt:hash_ancien)
-- KEYS[2]: new_key (rt:hash_nouveau)
-- ARGV[1]: new_token_json_partiel
-- ARGV[2]: ttl_ms
-- ARGV[3]: now_timestamp

local old_data = redis.call("GET", KEYS[1])
if not old_data then 
    return redis.error_reply("TOKEN_NOT_FOUND") 
end

local old_token = cjson.decode(old_data)

-- Résilience sur la casse Go (PascalCase vs snake_case)
local family_id = old_token.FamilyID or old_token.family_id
local used_at = old_token.UsedAt or old_token.used_at

if not family_id then
    return redis.error_reply("INTERNAL_ERROR: Missing FamilyID")
end

local family_key = "rt:family:" .. tostring(family_id)

-- 🛡️ DÉTECTION DE REPLAY (USAGE MULTIPLE)
if used_at ~= nil then
    -- 🚨 Alerte Sécurité : On révoque TOUTE la famille
    local members = redis.call("SMEMBERS", family_key)
    if #members > 0 then 
        redis.call("DEL", unpack(members)) 
    end
    redis.call("DEL", family_key)
    return redis.error_reply("REPLAY_DETECTED")
end

-- 🧬 FUSION & CRÉATION
local new_token = cjson.decode(ARGV[1])
new_token.FamilyID = family_id
new_token.UserID = old_token.UserID or old_token.user_id
new_token.TenantID = old_token.TenantID or old_token.tenant_id

-- ⏱️ INVALIDATION (Grace Period de 60s pour la latence réseau)
old_token.UsedAt = ARGV[3]
redis.call("SET", KEYS[1], cjson.encode(old_token), "PX", 60000)

-- 🚀 PERSISTENCE
local final_json = cjson.encode(new_token)
redis.call("SET", KEYS[2], final_json, "PX", ARGV[2])
redis.call("SADD", family_key, KEYS[2])
redis.call("PEXPIRE", family_key, ARGV[2]) -- La famille expire avec le dernier token

return final_json