package middlewares

// ContextKey est un type personnalisé obligatoire en Go pour éviter
// les collisions de clés entre différents packages dans un context.Context
type ContextKey string

const (
	TenantIDKey ContextKey = "tenant_id"
	UserIDKey   ContextKey = "user_id"
	RoleKey     ContextKey = "role"
)
