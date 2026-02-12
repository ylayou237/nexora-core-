🚀 PROMPT STRATÉGIQUE NEXORA-CORE V8.0 FINAL

Enterprise ISP/WISP Platform – Carrier Grade – Production Ready

🎯 CONTEXTE & MISSION

Tu es un Architecte Logiciel Senior spécialisé dans les plateformes ISP/WISP Carrier-Grade, expert en :

Go (Golang) production-grade pour systèmes critiques

Architecture hexagonale et microservices

RADIUS/AAA et gestion réseau télécoms

SaaS Multi-tenant B2B/B2B2C

Systèmes financiers sécurisés

Mission : Génère le module [NOM DU MODULE] pour Nexora-Core, une plateforme complète de gestion ISP/WISP avec :

Multi-tenant (Opérateur → Fournisseurs → Revendeurs → Clients)

Support 100k+ utilisateurs actifs

Disponibilité 99.95% (carrier-grade)

Conformité réglementaire (interception légale, RGPD, retention)

🏗 ARCHITECTURE GLOBALE

Structure Projet (Respecter STRICTEMENT)

nexora-core/
├── cmd/
│ ├── api/ # API REST principale
│ │ └── main.go
│ ├── radius-server/ # Serveur RADIUS dédié
│ │ └── main.go
│ ├── accounting-consumer/ # Consumer NATS accounting
│ │ └── main.go
│ └── worker/ # Jobs async (billing, notifications)
│ └── main.go
│
├── internal/
│ ├── adapters/
│ │ ├── primary/ # Driving Adapters
│ │ │ ├── radius/
│ │ │ │ ├── server.go # Serveur RADIUS (layeh.com/radius)
│ │ │ │ ├── auth_handler.go
│ │ │ │ ├── acct_handler.go
│ │ │ │ ├── coa_handler.go # Change of Authorization
│ │ │ │ └── drivers/ # Vendor-specific (VSA)
│ │ │ │ ├── mikrotik.go
│ │ │ │ ├── ubiquiti.go
│ │ │ │ ├── cisco.go
│ │ │ │ └── huawei.go
│ │ │ │
│ │ │ ├── web/
│ │ │ │ ├── server.go # HTTP/REST server
│ │ │ │ ├── middleware/
│ │ │ │ │ ├── auth.go # JWT validation
│ │ │ │ │ ├── rbac.go # Permission checks
│ │ │ │ │ ├── rate_limit.go
│ │ │ │ │ └── audit.go # Audit trail
│ │ │ │ ├── handlers/
│ │ │ │ │ ├── auth_handler.go
│ │ │ │ │ ├── user_handler.go
│ │ │ │ │ ├── billing_handler.go
│ │ │ │ │ ├── ipam_handler.go
│ │ │ │ │ ├── portal_handler.go
│ │ │ │ │ ├── gis_handler.go
│ │ │ │ │ └── crm_handler.go
│ │ │ │ └── dto/ # Data Transfer Objects
│ │ │ │
│ │ │ └── captive/
│ │ │ ├── portal_server.go # Portail captif
│ │ │ └── templates/ # Templates dynamiques
│ │ │
│ │ └── secondary/ # Driven Adapters
│ │ ├── postgres/
│ │ │ ├── adapter.go # Pool pgxpool
│ │ │ ├── user_repository.go
│ │ │ ├── plan_repository.go
│ │ │ ├── nas_repository.go
│ │ │ ├── billing_repository.go
│ │ │ ├── wallet_repository.go
│ │ │ ├── ipam_repository.go
│ │ │ ├── audit_repository.go
│ │ │ ├── portal_repository.go
│ │ │ ├── gis_repository.go
│ │ │ ├── crm_repository.go
│ │ │ └── transaction.go # Transaction manager
│ │ │
│ │ ├── redis/
│ │ │ ├── cache.go # Redis Sentinel client
│ │ │ ├── session.go # Session management
│ │ │ ├── ipam.go # IP pool management
│ │ │ └── rate_limiter.go
│ │ │
│ │ ├── queue/
│ │ │ └── nats.go # NATS JetStream
│ │ │
│ │ ├── payment/
│ │ │ ├── stripe.go # Stripe adapter
│ │ │ ├── paypal.go
│ │ │ ├── mobile_money.go
│ │ │ └── factory.go # Payment gateway factory
│ │ │
│ │ ├── notification/
│ │ │ ├── email.go # SMTP/SendGrid
│ │ │ ├── sms.go # Twilio/Nexmo
│ │ │ └── push.go # Firebase
│ │ │
│ │ ├── storage/
│ │ │ └── s3.go # Document storage
│ │ │
│ │ └── gis/
│ │ └── nominatim.go # Geocoding
│ │
│ └── core/
│ ├── domain/
│ │ ├── user.go # User entity + value objects
│ │ ├── tenant.go # Multi-tenant hierarchy
│ │ ├── plan.go
│ │ ├── nas.go
│ │ ├── session.go # RADIUS session
│ │ ├── wallet.go # Financial wallet
│ │ ├── transaction.go
│ │ ├── invoice.go
│ │ ├── ip_pool.go # IPAM
│ │ ├── ip_allocation.go
│ │ ├── portal_template.go # Captive portal
│ │ ├── gis_location.go # GIS entities
│ │ ├── lead.go # CRM lead
│ │ ├── contract.go # CRM contract
│ │ ├── audit_log.go
│ │ ├── errors.go # Sentinel errors
│ │ └── value_objects.go # Username, Email, MAC, etc.
│ │
│ ├── ports/
│ │ ├── repositories.go # Repository interfaces
│ │ ├── services.go # Service interfaces
│ │ ├── payment.go # Payment gateway interface
│ │ └── notification.go # Notification interface
│ │
│ └── services/
│ ├── auth_service.go # Authentication + RBAC
│ ├── radius_service.go # RADIUS business logic
│ ├── accounting_service.go
│ ├── billing_service.go # Invoicing + wallet
│ ├── payment_service.go # Multi-gateway
│ ├── ipam_service.go # IP management
│ ├── portal_service.go # Captive portal
│ ├── qos_service.go # QoS policies
│ ├── gis_service.go # GIS + eligibility
│ ├── crm_service.go # Lead management
│ ├── notification_service.go
│ ├── audit_service.go # Audit trail
│ ├── cache_service.go
│ └── isp_service.go # Provider/reseller logic
│
├── migrations/
│ ├── 001_initial_schema.up.sql
│ ├── 001_initial_schema.down.sql
│ ├── 002_add_billing.up.sql
│ ├── 002_add_billing.down.sql
│ ├── 003_add_ipam.up.sql
│ ├── 003_add_ipam.down.sql
│ ├── 004_add_gis.up.sql
│ ├── 004_add_gis.down.sql
│ ├── 005_add_crm.up.sql
│ └── 005_add_crm.down.sql
│
├── tests/
│ ├── unit/
│ │ ├── auth_service_test.go
│ │ ├── billing_service_test.go
│ │ └── ipam_service_test.go
│ ├── integration/
│ │ ├── radius_test.go
│ │ ├── payment_test.go
│ │ └── ipam_test.go
│ ├── e2e/
│ │ └── full_flow_test.go
│ └── benchmarks/
│ ├── auth_bench_test.go
│ └── accounting_bench_test.go
│
├── docs/
│ ├── ARCHITECTURE.md # C4 diagrams, ADR
│ ├── RUNBOOK.md # Incident procedures
│ ├── API.md # REST API documentation
│ ├── RADIUS.md # RADIUS attributes
│ ├── DEPLOYMENT.md # Deploy guide
│ ├── SECURITY.md # Security practices
│ └── COMPLIANCE.md # Legal requirements
│
├── deployments/
│ ├── docker-compose.yml # Dev environment
│ ├── docker-compose.prod.yml # Production stack
│ ├── kubernetes/
│ │ ├── namespace.yaml
│ │ ├── configmap.yaml
│ │ ├── secrets.yaml
│ │ ├── radius-deployment.yaml
│ │ ├── api-deployment.yaml
│ │ ├── worker-deployment.yaml
│ │ ├── consumer-deployment.yaml
│ │ ├── service.yaml
│ │ ├── ingress.yaml
│ │ ├── hpa.yaml # Horizontal Pod Autoscaler
│ │ └── monitoring/
│ │ ├── prometheus.yaml
│ │ └── grafana-dashboard.json
│ └── terraform/
│ ├── main.tf
│ ├── vpc.tf
│ ├── rds.tf
│ ├── elasticache.tf
│ └── eks.tf
│
├── configs/
│ ├── config.dev.yaml
│ ├── config.staging.yaml
│ └── config.prod.yaml
│
├── scripts/
│ ├── migrate.sh # Database migrations
│ ├── backup.sh # Automated backups
│ ├── failover.sh # DR failover
│ ├── seed_dev.sh # Dev data seeding
│ └── ci/
│ ├── test.sh
│ ├── build.sh
│ └── deploy.sh
│
├── web/ # Frontend PWA (optionnel)
│ ├── public/
│ ├── src/
│ │ ├── components/
│ │ ├── pages/
│ │ ├── services/
│ │ └── store/
│ └── package.json
│
├── Makefile
├── .golangci.yml
├── .env.example
├── go.mod
├── go.sum
├── Dockerfile
├── .dockerignore
├── .gitignore
└── README.md

🔐 SÉCURITÉ & CONFORMITÉ (CARRIER GRADE)

Authentication & Authorization

Multi-Factor Authentication:

Password: bcrypt cost=12

OTP: TOTP (RFC 6238) via Google Authenticator

SMS: Twilio/Nexmo avec rate limiting

Email: Verification links avec expiration

JWT Tokens:

Algorithme: RS256 (asymmetric)

Expiration: Access 15min, Refresh 7 jours

Revocation: Blacklist Redis avec TTL

Claims: user_id, tenant_id, role, permissions

RBAC (Role-Based Access Control):
Hiérarchie:
- SuperAdmin: Accès complet système
- TenantAdmin: Gestion tenant isolé
- Provider: Gestion revendeurs + clients
- Reseller: Gestion clients uniquement
- Customer: Accès profil + consommation
- Support: Read-only + actions limitées

Permissions granulaires:
- users.create, users.read, users.update, users.delete
- billing.read, billing.manage
- radius.disconnect, radius.change_plan
- audit.read (sensitive)
- ipam.allocate, ipam.release

Exemple implémentation RBAC :

package middleware

type Permission string

const (
PermUserCreate Permission = "users.create"
PermUserRead Permission = "users.read"
PermBillingManage Permission = "billing.manage"
PermRADIUSDisconnect Permission = "radius.disconnect"
)

var rolePermissions = map[string][]Permission{
"super_admin": {
PermUserCreate, PermUserRead, PermBillingManage, PermRADIUSDisconnect,
},
"provider": {
PermUserCreate, PermUserRead, PermBillingManage,
},
"reseller": {
PermUserRead,
},
"customer": {},
}

func RequirePermission(perm Permission) gin.HandlerFunc {
return func(c *gin.Context) {
claims := c.MustGet("claims").(*domain.TokenClaims)

if !hasPermission(claims.Role, perm) { c.JSON(403, gin.H{"error": "insufficient permissions"}) c.Abort() return } c.Next() } 

}

Audit Trail (Immuable)

Stockage:

Table: audit_logs (append-only)

Partition: Par mois (PostgreSQL partitioning)

Retention: 7 ans (conformité légale)

Immutabilité: Pas de UPDATE/DELETE

Événements loggés:

Authentification: login, logout, failed_login

Financier: payment, refund, wallet_debit

RADIUS: session_start, session_stop, disconnect

Admin: user_create, user_delete, plan_change

IPAM: ip_allocate, ip_release

Système: config_change, backup_started

Format:

timestamp (nanosecond precision)

user_id, tenant_id, ip_address

action, resource_type, resource_id

old_value, new_value (JSON)

correlation_id (trace_id)

Schema SQL :

CREATE TABLE audit_logs (
id BIGSERIAL,
timestamp TIMESTAMP(6) NOT NULL DEFAULT NOW(),
user_id INTEGER,
tenant_id INTEGER NOT NULL,
ip_address INET,
action VARCHAR(50) NOT NULL,
resource_type VARCHAR(50) NOT NULL,
resource_id VARCHAR(255),
old_value JSONB,
new_value JSONB,
correlation_id UUID,
PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

CREATE INDEX idx_audit_tenant_time ON audit_logs(tenant_id, timestamp DESC);
CREATE INDEX idx_audit_user_action ON audit_logs(user_id, action);

-- Partition mensuelle automatique
CREATE TABLE audit_logs_2025_02 PARTITION OF audit_logs
FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

Interception Légale (Lawful Intercept)

Conformité:

CALEA (USA), ETSI (Europe), lois locales

Targets identifiés par autorité judiciaire

Logs immutables et chiffrés

Accès strictement restreint

Fonctionnalités:

Capture sessions RADIUS (username, IP, dates)

Export trafic réseau (pcap) via mirroring

Blocage administratif immédiat

Rapports périodiques autorités

Sécurité:

Clé chiffrement hardware (HSM)

Dual control: 2 admin pour activation

Audit trail dédié (qui a activé quand)

Exemple service :

package services

type LawfulInterceptService struct {
repo ports.InterceptRepository
radius ports.RADIUSService
}

func (s *LawfulInterceptService) ActivateTarget(
ctx context.Context,
username string,
courtOrder string,
expiresAt time.Time,
) error {
// Validation dual control
if !s.validateDualControl(ctx) {
return errors.New("dual control required")
}

target := &domain.InterceptTarget{ Username: username, CourtOrder: courtOrder, // Stocké chiffré ActivatedAt: time.Now(), ExpiresAt: expiresAt, Status: "active", } // Enregistrer dans table sécurisée if err := s.repo.Create(ctx, target); err != nil { return err } // Logger audit (immuable) s.auditLog(ctx, "lawful_intercept_activated", target) // Activer capture réseau (hors scope Go) // Appel API équipement réseau pour mirroring port return nil 

}

Protection DDoS & Rate Limiting

Layers:

Reverse Proxy: Cloudflare/Nginx avec limite globale

API Gateway: Kong/Traefik avec quota par tenant

Application: Redis-based token bucket

RADIUS: UDP stateless avec cache NAS

Limites par défaut:

API REST: 100 req/min/IP, 1000 req/min/tenant

RADIUS Auth: 50 auth/s/NAS

Payment: 10 transactions/min/user

Captive Portal: 20 logins/min/IP

Détection anomalies:

Spike auth failures (> 50/min) → alerte

Payment fraud (montants suspects) → blocage

Brute force login → captcha + ban temporaire

💳 GESTION FINANCIÈRE (BILLING & PAYMENTS)

Architecture Wallet Multi-Devise

Entités:

Wallet: Compte virtuel par utilisateur/revendeur

Transaction: Mouvement atomique (debit/credit)

Invoice: Facture générée automatiquement

Commission: Partage revenus hiérarchique

Fonctionnalités:

Multi-currency: EUR, USD, XOF, etc.

Decimal precision: 4 décimales (crypto-ready)

Double-entry bookkeeping (comptabilité)

Atomic transactions: ACID garanti

Idempotence: UUID transaction pour retry safe

Balance cache: Redis pour performance

Schema SQL :

CREATE TABLE wallets (
id SERIAL PRIMARY KEY,
user_id INTEGER NOT NULL REFERENCES users(id),
currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
balance DECIMAL(18,4) NOT NULL DEFAULT 0 CHECK (balance >= 0),
locked_balance DECIMAL(18,4) NOT NULL DEFAULT 0,
created_at TIMESTAMP DEFAULT NOW(),
updated_at TIMESTAMP DEFAULT NOW(),

CONSTRAINT uk_user_currency UNIQUE(user_id, currency) 

);

CREATE TABLE transactions (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
wallet_id INTEGER NOT NULL REFERENCES wallets(id),
type VARCHAR(20) NOT NULL, -- credit, debit, commission
amount DECIMAL(18,4) NOT NULL,
balance_after DECIMAL(18,4) NOT NULL,
reference VARCHAR(255), -- Payment gateway ref
description TEXT,
metadata JSONB,
created_at TIMESTAMP DEFAULT NOW(),
idempotency_key VARCHAR(255) UNIQUE,

CHECK (amount > 0) 

);

CREATE INDEX idx_transactions_wallet_time ON transactions(wallet_id, created_at DESC);

CREATE TABLE invoices (
id SERIAL PRIMARY KEY,
tenant_id INTEGER NOT NULL,
user_id INTEGER NOT NULL,
invoice_number VARCHAR(50) NOT NULL UNIQUE,
amount DECIMAL(10,2) NOT NULL,
currency VARCHAR(3) NOT NULL,
status VARCHAR(20) NOT NULL, -- pending, paid, cancelled
due_date DATE NOT NULL,
paid_at TIMESTAMP,
pdf_url TEXT,
created_at TIMESTAMP DEFAULT NOW()
);

Multi-Gateway Payment

package payment

type Gateway interface {
CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error)
VerifyWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
Refund(ctx context.Context, paymentID string, amount decimal.Decimal) error
}

type PaymentRequest struct {
Amount decimal.Decimal
Currency string
UserID uint
Description string
ReturnURL string
WebhookURL string
Metadata map[string]string
}

// Factory pattern
func NewGateway(provider string, config Config) (Gateway, error) {
switch provider {
case "stripe":
return NewStripeGateway(config.StripeSecretKey), nil
case "paypal":
return NewPayPalGateway(config.PayPalClientID, config.PayPalSecret), nil
case "mobile_money":
return NewMobileMoneyGateway(config.MMAPI), nil
default:
return nil, fmt.Errorf("unknown provider: %s", provider)
}
}

Commission & Revenue Sharing

Hiérarchie:
Opérateur (100%)
→ Fournisseur (80%)
→ Revendeur (60%)
→ Client final

Exemple:

Client paie 100 EUR

Revendeur reçoit 60 EUR (commission 60%)

Fournisseur reçoit 20 EUR (80% - 60%)

Opérateur reçoit 20 EUR (100% - 80%)

Calcul automatique:

Lors du paiement client

Crédit wallet en cascade

Transaction atomique (ACID)

Implémentation :

func (s *BillingService) ProcessPayment(ctx context.Context, payment *Payment) error {
// 1. Démarrer transaction DB
tx, _ := s.db.BeginTx(ctx, nil)
defer tx.Rollback()

// 2. Créditer wallet client s.walletRepo.Credit(ctx, tx, payment.UserID, payment.Amount) // 3. Calculer et distribuer commissions hierarchy := s.getHierarchy(ctx, payment.UserID) for _, level := range hierarchy { commission := payment.Amount.Mul(level.CommissionRate) s.walletRepo.Credit(ctx, tx, level.UserID, commission) s.transactionRepo.Create(ctx, tx, &Transaction{ Type: "commission", Amount: commission, Description: fmt.Sprintf("Commission from %s", payment.UserID), }) } // 4. Commit return tx.Commit() 

}

Facturation Automatique

Déclencheurs:

Fin de période (mensuel/annuel)

Consommation quota (pay-as-you-go)

Accounting RADIUS (data consumed)

Processus:

Calculer montant selon plan + consommation

Générer invoice (PDF via HTML template)

Débit wallet ou appel payment gateway

Envoi email + push notification

Si échec: grace period 3 jours → suspension

Template Invoice:

Logo revendeur

Détail consommation (data, temps)

TVA applicable selon pays

Mentions légales obligatoires

🌐 RADIUS & ACCOUNTING (Carrier Grade)

Isolation Stricte Multi-Tenant

Principe:

Chaque requête RADIUS validée par NAS IP

NAS lié à 1 tenant unique

User lookup scopé par tenant_id

Impossible d'authentifier user d'un autre tenant

Sécurité:

Shared secret unique par NAS (Argon2id)

Message-Authenticator obligatoire

IP whitelist stricte

Rate limiting par NAS

Anti-Clonage & Session Unicity

MAC Binding (Sticky):

Lors première auth: MAC → User binding

Auths suivantes: validation MAC == binding

Si différent: reject avec raison "DEVICE_MISMATCH"

Option configurable par tenant

Session Limits:

Max 1 session simultanée (défaut)

Option multi-device pour entreprises (max 5)

Détection via Accounting Start

Auto-disconnect si limite atteinte

Implémentation :

func (s *RADIUSService) CheckSessionLimit(ctx context.Context, user *User) error {
activeSessions := s.repo.CountActiveSessions(ctx, user.ID)

if activeSessions >= user.MaxSessions { // Déconnecter session la plus ancienne oldestSession := s.repo.GetOldestSession(ctx, user.ID) s.sendDisconnect(ctx, oldestSession) return errors.New("session limit reached, disconnected oldest") } return nil 

}

Accounting Asynchrone (NATS JetStream)

Architecture:
RADIUS Handler → NATS Queue → Consumer Pool → DB Batch Insert

Avantages:

Ack immédiat au NAS (< 10ms)

Pas de blocage si DB lente

Résilience: retry automatique

Scalabilité: N consumers parallèles

Pattern:

RADIUS reçoit Accounting-Request

Publish message NATS (idempotence key = Acct-Unique-ID)

Return Accounting-Response au NAS

Consumer pull batch 100 messages

Upsert DB (GREATEST pour éviter régression)

Ack messages après succès

Configuration NATS:

Stream: accounting.sessions

Retention: 7 jours

Replicas: 3 (HA)

Consumer group: accounting-writers

Max ack pending: 1000

CoA & Disconnect (RFC 3576)

Use Cases:

Changement plan en temps réel

Suspension pour impayé

Ajustement vitesse (throttling)

Déconnexion administrateur

Implémentation:

Packet CoA-Request vers NAS:3799/udp

Attributs: Session-ID, Speed-Limit, Action

Retry 3x avec backoff si timeout

Circuit breaker si NAS down

Exemple :

func (s *RADIUSService) ChangePlan(ctx context.Context, userID uint, newPlanID uint) error {
user := s.userRepo.GetByID(ctx, userID)
plan := s.planRepo.GetByID(ctx, newPlanID)

// Update DB s.userRepo.UpdatePlan(ctx, userID, newPlanID) // Si session active session := s.getActiveSession(ctx, userID) if session != nil { // Envoyer CoA pour changer vitesse return s.sendCoA(ctx, &CoARequest{ NASIP: session.NASIP, SessionID: session.RadiusSessionID, SpeedLimit: plan.SpeedLimit, }) } return nil 

}

🌍 IPAM (IP Address Management)

Architecture Dual-Stack (IPv4 + IPv6)

Pools:

IPv4: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16

IPv6: 2001:db8::/32 (exemple, adapté au préfixe ISP)

Types d'allocation:

Dynamic: DHCP-like, libérée à déconnexion

Static: IP fixe pour clients Business

Sticky: Même IP si reconnexion < 24h

Prefix Delegation: /56 ou /64 pour clients FTTH

Exclusions:

Network address (.0)

Broadcast (.255)

Gateway (premier IP)

DNS servers (configurables)

Schema SQL :

CREATE TABLE ip_pools (
id SERIAL PRIMARY KEY,
tenant_id INTEGER NOT NULL,
name VARCHAR(100) NOT NULL,
cidr CIDR NOT NULL,
gateway INET,
dns_primary INET,
dns_secondary INET,
version INTEGER NOT NULL CHECK (version IN (4, 6)),
allocation_type VARCHAR(20) NOT NULL, -- dynamic, static, sticky
created_at TIMESTAMP DEFAULT NOW(),

CONSTRAINT uk_tenant_cidr UNIQUE(tenant_id, cidr) 

);

CREATE TABLE ip_allocations (
id SERIAL PRIMARY KEY,
pool_id INTEGER NOT NULL REFERENCES ip_pools(id),
ip_address INET NOT NULL,
user_id INTEGER REFERENCES users(id),
mac_address MACADDR,
session_id VARCHAR(255),
allocated_at TIMESTAMP DEFAULT NOW(),
released_at TIMESTAMP,
status VARCHAR(20) NOT NULL, -- allocated, reserved, released

CONSTRAINT uk_pool_ip UNIQUE(pool_id, ip_address) 

);

CREATE INDEX idx_ip_alloc_user ON ip_allocations(user_id) WHERE released_at IS NULL;
CREATE INDEX idx_ip_alloc_pool_status ON ip_allocations(pool_id, status);

Allocation via Redis (Performance)

Stratégie:

Pool IP stocké en Redis SET

Allocation = SPOP (atomic)

Libération = SADD

Persistence DB asynchrone

Avantages:

Latence < 1ms (vs 10-50ms DB)

Race-condition free (Redis single-threaded)

Scalabilité horizontale RADIUS servers

Synchronisation:

Warmup: Load pool depuis DB → Redis au démarrage

Accounting Stop: Libération Redis + DB

Reconciliation job: Toutes les 5min sync Redis ↔ DB

Implémentation :

package ipam

type RedisIPAM struct {
redis *redis.Client
repo ports.IPAMRepository
}

func (r *RedisIPAM) AllocateIP(ctx context.Context, poolID uint, userID uint) (net.IP, error) {
key := fmt.Sprintf("pool:%d:available", poolID)

// Atomic pop from set ipStr, err := r.redis.SPop(ctx, key).Result() if err != nil { // Pool exhausted return nil, errors.New("no IP available") } ip := net.ParseIP(ipStr) // Async persist to DB go r.repo.CreateAllocation(context.Background(), &domain.IPAllocation{ PoolID: poolID, IPAddress: ip, UserID: userID, Status: "allocated", }) return ip, nil 

}

func (r *RedisIPAM) ReleaseIP(ctx context.Context, ip net.IP, poolID uint) error {
key := fmt.Sprintf("pool:%d:available", poolID)

// Return to pool r.redis.SAdd(ctx, key, ip.String()) // Update DB return r.repo.ReleaseAllocation(ctx, ip) 

}

IPv6 Prefix Delegation

Use Case:

Client FTTH reçoit un /56 ou /64

Sous-réseau pour LAN client

Routage délégué

Implémentation:

Pool /48 divisé en préfixes /56

Attribution via DHCPv6-PD

Stockage allocation en DB

Libération à résiliation

🎨 PORTAIL CAPTIF DYNAMIQUE

Templates Multi-Tenant

Architecture:

Templates HTML/CSS stockés en DB

Variables: {{logo}}, {{title}}, {{tos_url}}, {{promo}}

Rendering: Go html/template

Cache: Redis par tenant_id

Personnalisation revendeur:

Logo (URL S3)

Couleurs primaire/secondaire (hex)

Texte accueil, CGU, messages

Bannière publicitaire

Mode: User/Pass, SMS OTP, Social, Free+Ads

Modes d'authentification:

Classic: Username/Password

SMS OTP: Numéro → code → session temporaire

Social: Google/Facebook OAuth2

Gratuit avec pub: 30min gratuit après vidéo pub

Schema SQL :

CREATE TABLE portal_templates (
id SERIAL PRIMARY KEY,
tenant_id INTEGER NOT NULL,
name VARCHAR(100) NOT NULL,
logo_url TEXT,
primary_color VARCHAR(7) DEFAULT '#007bff',
secondary_color VARCHAR(7) DEFAULT '#6c757d',
welcome_text TEXT,
tos_url TEXT,
ad_banner_html TEXT,
auth_modes TEXT[] NOT NULL, -- {password, sms, social, free}
html_template TEXT NOT NULL,
css_template TEXT,
created_at TIMESTAMP DEFAULT NOW(),
updated_at TIMESTAMP DEFAULT NOW(),

CONSTRAINT uk_tenant_template UNIQUE(tenant_id, name) 

);

Rendu Dynamique

package captive

type PortalHandler struct {
service ports.PortalService
cache ports.CacheService
}

func (h *PortalHandler) RenderPortal(c *gin.Context) {
nasIP := c.Query("nas_ip")

// Get tenant from NAS nas := h.service.GetNASByIP(nasIP) // Get template (cache first) template := h.cache.GetPortalTemplate(nas.TenantID) if template == nil { template = h.service.GetActiveTemplate(nas.TenantID) h.cache.SetPortalTemplate(nas.TenantID, template, 10*time.Minute) } // Render data := map[string]interface{}{ "Logo": template.LogoURL, "Title": template.WelcomeText, "PrimaryColor": template.PrimaryColor, "AuthModes": template.AuthModes, } tmpl := html.Must(html.New("portal").Parse(template.HTMLTemplate)) tmpl.Execute(c.Writer, data) 

}

📍 GIS & CARTOGRAPHIE

Entités Géographiques

Modèles:

POP (Point of Presence): Tour/site avec équipements

Coverage Zone: Polygone zone couverte

Customer Location: Point GPS client

Intervention: Géolocalisation technicien

Fonctionnalités:

Visualisation carte (Leaflet/Mapbox)

Calcul éligibilité (distance, LoS, Fresnel)

Clustering clients par zone

Heatmap densité abonnés

Routing technicien (optimisation trajet)

Schema SQL :

CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE pops (
id SERIAL PRIMARY KEY,
tenant_id INTEGER NOT NULL,
name VARCHAR(100) NOT NULL,
location GEOGRAPHY(POINT, 4326) NOT NULL,
altitude INTEGER, -- mètres
height_agl INTEGER, -- hauteur antenne
capacity_mbps INTEGER,
status VARCHAR(20) NOT NULL,
created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_pops_location ON pops USING GIST(location);

CREATE TABLE coverage_zones (
id SERIAL PRIMARY KEY,
pop_id INTEGER NOT NULL REFERENCES pops(id),
zone GEOGRAPHY(POLYGON, 4326) NOT NULL,
signal_strength INTEGER, -- dBm
technology VARCHAR(20), -- 2.4GHz, 5GHz, 60GHz
created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_coverage_zone ON coverage_zones USING GIST(zone);

CREATE TABLE customer_locations (
id SERIAL PRIMARY KEY,
user_id INTEGER NOT NULL REFERENCES users(id),
location GEOGRAPHY(POINT, 4326) NOT NULL,
address TEXT,
building_height INTEGER,
created_at TIMESTAMP DEFAULT NOW()
);

Calcul Éligibilité

package gis

type EligibilityService struct {
repo ports.GISRepository
}

type EligibilityResult struct {
Eligible bool
POPs []POP
Distance float64 // km
LineOfSight bool
FresnelZone float64 // % clearance
EstimatedSpeed int // Mbps
}

func (s *EligibilityService) CheckEligibility(
ctx context.Context,
location *geo.Point,
) (*EligibilityResult, error) {

// 1. Trouver POPs dans rayon 10km pops := s.repo.FindPOPsWithinRadius(ctx, location, 10000) if len(pops) == 0 { return &EligibilityResult{Eligible: false}, nil } // 2. Pour chaque POP, calculer LoS et Fresnel for _, pop := range pops { distance := geo.Distance(location, pop.Location) // Line of Sight (nécessite DEM - Digital Elevation Model) los := s.calculateLoS(location, pop.Location) // Fresnel zone clearance fresnel := s.calculateFresnel(location, pop.Location, 5.8) // 5.8 GHz if los && fresnel > 0.6 { return &EligibilityResult{ Eligible: true, POPs: []POP{pop}, Distance: distance / 1000, LineOfSight: true, FresnelZone: fresnel, EstimatedSpeed: s.estimateSpeed(distance, fresnel), }, nil } } return &EligibilityResult{Eligible: false}, nil 

}

📞 CRM & LEAD MANAGEMENT

Pipeline Conversion

Étapes:

Lead: Prospect depuis formulaire éligibilité

Qualified: Éligible confirmé, devis envoyé

Proposal: Proposition commerciale en cours

Contract: Contrat signé électroniquement

Customer: Client actif

Automatisations:

Email accueil après lead création

Relance si pas de réponse 7 jours

Assignation technicien pour installation

Activation compte post-installation

Signature Électronique:

Intégration DocuSign/HelloSign

Template contrat avec variables

Validation juridique

Schema SQL :

CREATE TABLE leads (
id SERIAL PRIMARY KEY,
tenant_id INTEGER NOT NULL,
first_name VARCHAR(100) NOT NULL,
last_name VARCHAR(100) NOT NULL,
email VARCHAR(255) NOT NULL,
phone VARCHAR(20),
address TEXT,
location GEOGRAPHY(POINT, 4326),
status VARCHAR(20) NOT NULL, -- lead, qualified, proposal, contract, customer, lost
source VARCHAR(50), -- website, referral, campaign
assigned_to INTEGER REFERENCES users(id),
eligibility JSONB, -- Résultat calcul éligibilité
created_at TIMESTAMP DEFAULT NOW(),
updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE contracts (
id SERIAL PRIMARY KEY,
lead_id INTEGER NOT NULL REFERENCES leads(id),
contract_number VARCHAR(50) NOT NULL UNIQUE,
plan_id INTEGER NOT NULL REFERENCES plans(id),
start_date DATE NOT NULL,
end_date DATE,
signed_at TIMESTAMP,
signature_url TEXT, -- DocuSign PDF
status VARCHAR(20) NOT NULL, -- draft, sent, signed, active, terminated
created_at TIMESTAMP DEFAULT NOW()
);

Service CRM

package services

type CRMService struct {
leadRepo ports.LeadRepository
contractRepo ports.ContractRepository
notification ports.NotificationService
docusign ports.DocumentSignService
}

func (s *CRMService) CreateLeadFromEligibility(
ctx context.Context,
req *CreateLeadRequest,
) (*domain.Lead, error) {

// 1. Vérifier éligibilité eligibility := s.gisService.CheckEligibility(ctx, req.Location) lead := &domain.Lead{ TenantID: req.TenantID, FirstName: req.FirstName, LastName: req.LastName, Email: req.Email, Phone: req.Phone, Location: req.Location, Status: "lead", Eligibility: eligibility, } // 2. Créer lead if err := s.leadRepo.Create(ctx, lead); err != nil { return nil, err } // 3. Email accueil s.notification.SendEmail(ctx, &EmailRequest{ To: lead.Email, Subject: "Bienvenue chez [Opérateur]", Template: "welcome_lead", Data: map[string]interface{}{"Lead": lead}, }) // 4. Si éligible, auto-qualify if eligibility.Eligible { lead.Status = "qualified" s.leadRepo.Update(ctx, lead) // Assigner commercial s.assignSalesRep(ctx, lead) } return lead, nil 

}

func (s *CRMService) GenerateContract(
ctx context.Context,
leadID uint,
planID uint,
) (*domain.Contract, error) {

lead := s.leadRepo.GetByID(ctx, leadID) plan := s.planRepo.GetByID(ctx, planID) // Générer numéro contrat contractNumber := s.generateContractNumber(lead.TenantID) contract := &domain.Contract{ LeadID: leadID, ContractNumber: contractNumber, PlanID: planID, StartDate: time.Now().AddDate(0, 0, 7), // +7 jours Status: "draft", } s.contractRepo.Create(ctx, contract) // Envoyer pour signature électronique signatureURL, _ := s.docusign.SendForSignature(ctx, &SignatureRequest{ RecipientEmail: lead.Email, RecipientName: fmt.Sprintf("%s %s", lead.FirstName, lead.LastName), DocumentData: map[string]interface{}{ "ContractNumber": contractNumber, "Plan": plan.Name, "Price": plan.Price, }, Template: "isp_contract_template", }) contract.SignatureURL = signatureURL contract.Status = "sent" s.contractRepo.Update(ctx, contract) return contract, nil 

}

📊 OBSERVABILITÉ & MONITORING

Métriques Prometheus (Complètes)

package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
// RADIUS
radiusAuthTotal = prometheus.NewCounterVec(
prometheus.CounterOpts{
Name: "radius_auth_total",
Help: "Total RADIUS authentication attempts",
},
[]string{"tenant", "result", "reason"},
)

radiusAuthDuration = prometheus.NewHistogramVec( prometheus.HistogramOpts{ Name: "radius_auth_duration_seconds", Help: "RADIUS authentication latency", Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1}, }, []string{"tenant"}, ) radiusActiveSessions = prometheus.NewGaugeVec( prometheus.GaugeOpts{ Name: "radius_active_sessions", Help: "Current active RADIUS sessions", }, []string{"tenant", "nas"}, ) radiusAccountingBytes = prometheus.NewCounterVec( prometheus.CounterOpts{ Name: "radius_accounting_bytes_total", Help: "Total bytes transferred (in/out)", }, []string{"tenant", "direction"}, ) // Billing paymentTotal = prometheus.NewCounterVec( prometheus.CounterOpts{ Name: "payment_total", Help: "Total payment attempts", }, []string{"tenant", "gateway", "status"}, ) paymentAmount = prometheus.NewHistogramVec( prometheus.HistogramOpts{ Name: "payment_amount_eur", Help: "Payment amount distribution", Buckets: []float64{5, 10, 20, 50, 100, 200, 500}, }, []string{"tenant", "gateway"}, ) walletBalance = prometheus.NewGaugeVec( prometheus.GaugeOpts{ Name: "wallet_balance_eur", Help: "Current wallet balance", }, []string{"tenant", "user_id"}, ) // IPAM ipPoolUtilization = prometheus.NewGaugeVec( prometheus.GaugeOpts{ Name: "ipam_pool_utilization_percent", Help: "IP pool utilization percentage", }, []string{"tenant", "pool_id"}, ) // Infrastructure dbPoolConns = prometheus.NewGaugeVec( prometheus.GaugeOpts{ Name: "db_pool_connections", Help: "Database connection pool state", }, []string{"state"}, // idle, active, waiting ) cacheHitRatio = prometheus.NewGaugeVec( prometheus.GaugeOpts{ Name: "cache_hit_ratio", Help: "Redis cache hit ratio", }, []string{"cache_type"}, ) queueDepth = prometheus.NewGaugeVec( prometheus.GaugeOpts{ Name: "queue_depth", Help: "NATS queue message pending", }, []string{"stream", "consumer"}, ) 

)

func init() {
prometheus.MustRegister(
radiusAuthTotal,
radiusAuthDuration,
radiusActiveSessions,
radiusAccountingBytes,
paymentTotal,
paymentAmount,
walletBalance,
ipPoolUtilization,
dbPoolConns,
cacheHitRatio,
queueDepth,
)
}

Dashboards Grafana

Dashboards:

RADIUS Operations:

Auth success rate (gauge)

Auth latency heatmap (p50/p95/p99)

Active sessions timeline

Accounting bytes (stacked area)

Top NAS by traffic

Billing & Finance:

Revenue today/week/month (stat)

Payment success rate

Top paying customers

Wallet balances distribution

Commission payouts

Infrastructure:

DB pool saturation

Redis hit ratio

Queue lag (NATS)

CPU/Memory per service

Network I/O

Business:

New signups timeline

Churn rate

ARPU (Average Revenue Per User)

Support ticket volume

GIS coverage heatmap

Alerting (PagerDuty/Opsgenie)

Critical Alerts:

Auth failure rate > 10% (5min) → Page on-call

DB pool exhausted > 2min → Page DBA

Payment gateway down > 1min → Page billing team

RADIUS server down > 30s → Page network team

IPAM pool > 90% full → Email network ops

Warning Alerts:

Auth latency p95 > 100ms → Slack channel

Queue lag > 10k messages → Slack channel

Disk usage > 80% → Email sysadmin

Unusual traffic spike → Slack security

Business Alerts:

Daily revenue < threshold → Email finance

Churn rate spike → Email customer success

New contract signed → Slack sales

🧪 TESTS & QUALITÉ

Strategy de Tests

Layers:

Unit Tests (80%+ coverage):

Domain logic pure

Services avec mocks

Value objects validation

Integration Tests:

Repository avec testcontainers

RADIUS packet flow

Payment gateway sandbox

E2E Tests:

User signup → payment → activation → session

Admin create reseller → reseller create customer

Performance Tests:

k6 load testing (1000 auth/s)

DB query benchmarks

Memory leak detection

Security Tests:

OWASP ZAP scanning

SQL injection attempts

JWT tampering

Rate limit bypass

Exemple Test Unitaire

package services_test

func TestBillingService_ProcessPayment(t *testing.T) {
tests := []struct {
name string
payment *domain.Payment
setupMocks func(*mocks.MockWalletRepo, *mocks.MockPaymentGateway)
wantErr bool
wantWalletCredit decimal.Decimal
}{
{
name: "successful payment with commission",
payment: &domain.Payment{
UserID: 1,
Amount: decimal.NewFromInt(100),
Currency: "EUR",
Gateway: "stripe",
},
setupMocks: func(repo *mocks.MockWalletRepo, gw *mocks.MockPaymentGateway) {
gw.EXPECT().
CreatePayment(gomock.Any(), gomock.Any()).
Return(&PaymentResponse{Status: "success"}, nil)

repo.EXPECT(). Credit(gomock.Any(), gomock.Any(), uint(1), decimal.NewFromInt(100)). Return(nil) // Commission revendeur (60%) repo.EXPECT(). Credit(gomock.Any(), gomock.Any(), uint(2), decimal.NewFromInt(60)). Return(nil) }, wantErr: false, wantWalletCredit: decimal.NewFromInt(100), }, { name: "payment gateway failure", payment: &domain.Payment{ UserID: 1, Amount: decimal.NewFromInt(50), Gateway: "stripe", }, setupMocks: func(repo *mocks.MockWalletRepo, gw *mocks.MockPaymentGateway) { gw.EXPECT(). CreatePayment(gomock.Any(), gomock.Any()). Return(nil, errors.New("gateway timeout")) }, wantErr: true, }, } for _, tt := range tests { t.Run(tt.name, func(t *testing.T) { ctrl := gomock.NewController(t) defer ctrl.Finish() mockRepo := mocks.NewMockWalletRepo(ctrl) mockGateway := mocks.NewMockPaymentGateway(ctrl) tt.setupMocks(mockRepo, mockGateway) service := services.NewBillingService(mockRepo, mockGateway) err := service.ProcessPayment(context.Background(), tt.payment) if (err != nil) != tt.wantErr { t.Errorf("ProcessPayment() error = %v, wantErr %v", err, tt.wantErr) } }) } 

}

Integration Test (Testcontainers)

package integration_test

func TestUserRepository_MultiTenant(t *testing.T) {
// Setup PostgreSQL container
ctx := context.Background()

postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ ContainerRequest: testcontainers.ContainerRequest{ Image: "postgres:15-alpine", ExposedPorts: []string{"5432/tcp"}, Env: map[string]string{ "POSTGRES_USER": "test", "POSTGRES_PASSWORD": "test", "POSTGRES_DB": "nexora_test", }, WaitingFor: wait.ForListeningPort("5432/tcp"), }, Started: true, }) require.NoError(t, err) defer postgres.Terminate(ctx) // Get connection host, _ := postgres.Host(ctx) port, _ := postgres.MappedPort(ctx, "5432") dsn := fmt.Sprintf("postgres://test:test@%s:%s/nexora_test", host, port.Port()) pool, _ := pgxpool.New(ctx, dsn) defer pool.Close() // Run migrations runMigrations(pool) repo := postgres.NewUserRepository(pool) // Test: Users from different tenants are isolated user1 := &domain.User{Username: "alice@tenant1", TenantID: 1} user2 := &domain.User{Username: "alice@tenant2", TenantID: 2} repo.Create(ctx, user1) repo.Create(ctx, user2) // Fetch tenant 1 retrieved, err := repo.GetByUsername(ctx, 1, "alice@tenant1") require.NoError(t, err) assert.Equal(t, uint(1), retrieved.TenantID) // Fetch tenant 2 retrieved, err = repo.GetByUsername(ctx, 2, "alice@tenant2") require.NoError(t, err) assert.Equal(t, uint(2), retrieved.TenantID) // Cross-tenant fetch should fail _, err = repo.GetByUsername(ctx, 1, "alice@tenant2") assert.Error(t, err) 

}

🚀 DEPLOYMENT & CI/CD

Docker Multi-Stage

Build stage

FROM golang:1.22-alpine AS builder

WORKDIR /app

Dependencies

COPY go.mod go.sum ./
RUN go mod download

Build

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
-ldflags="-w -s" \
-o /nexora-api ./cmd/api

Runtime stage

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /nexora-api .
COPY configs/ ./configs/
COPY migrations/ ./migrations/

EXPOSE 8080 1812/udp 1813/udp

CMD ["./nexora-api"]

Kubernetes Manifests

deployments/kubernetes/api-deployment.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
name: nexora-api
namespace: nexora
spec:
replicas: 3
selector:
matchLabels:
app: nexora-api
template:
metadata:
labels:
app: nexora-api
spec:
containers:
- name: api
image: nexora/api:v1.0.0
ports:
- containerPort: 8080
env:
- name: DB_HOST
valueFrom:
secretKeyRef:
name: nexora-secrets
key: db-host
- name: DB_PASSWORD
valueFrom:
secretKeyRef:
name: nexora-secrets
key: db-password
- name: REDIS_ADDR
value: "redis-sentinel:26379"
- name: NATS_URL
value: "nats://nats:4222"
resources:
requests:
memory: "512Mi"
cpu: "500m"
limits:
memory: "1Gi"
cpu: "1000m"
livenessProbe:
httpGet:
path: /health
port: 8080
initialDelaySeconds: 30
periodSeconds: 10
readinessProbe:
httpGet:
path: /ready
port: 8080
initialDelaySeconds: 10
periodSeconds: 5

apiVersion: v1
kind: Service
metadata:
name: nexora-api
namespace: nexora
spec:
type: LoadBalancer
ports:

port: 80
targetPort: 8080
name: http
selector:
app: nexora-api

HPA (Horizontal Pod Autoscaler)

apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
name: nexora-api-hpa
namespace: nexora
spec:
scaleTargetRef:
apiVersion: apps/v1
kind: Deployment
name: nexora-api
minReplicas: 3
maxReplicas: 10
metrics:

type: Resource
resource:
name: cpu
target:
type: Utilization
averageUtilization: 70

type: Resource
resource:
name: memory
target:
type: Utilization
averageUtilization: 80

CI/CD Pipeline (GitHub Actions)

.github/workflows/ci-cd.yml

name: CI/CD Pipeline

on:
push:
branches: [main, develop]
pull_request:
branches: [main]

jobs:
test:
runs-on: ubuntu-latest
steps:
- uses: actions/checkout@v3

- name: Setup Go uses: actions/setup-go@v4 with: go-version: '1.22' - name: Install dependencies run: go mod download - name: Run tests run: | go test -v -race -coverprofile=coverage.out ./... go tool cover -func=coverage.out - name: Run linter run: | go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest golangci-lint run - name: Security scan run: | go install github.com/securego/gosec/v2/cmd/gosec@latest gosec ./... 

build:
needs: test
runs-on: ubuntu-latest
if: github.ref == 'refs/heads/main'
steps:
- uses: actions/checkout@v3

- name: Build Docker image run: | docker build -t nexora/api:${{ github.sha }} . docker tag nexora/api:${{ github.sha }} nexora/api:latest - name: Push to registry run: | echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin docker push nexora/api:${{ github.sha }} docker push nexora/api:latest 

deploy:
needs: build
runs-on: ubuntu-latest
if: github.ref == 'refs/heads/main'
steps:
- name: Deploy to Kubernetes
run: |
kubectl set image deployment/nexora-api \
api=nexora/api:${{ github.sha }} \
--namespace=nexora \
--record

- name: Verify deployment run: | kubectl rollout status deployment/nexora-api -n nexora kubectl get pods -n nexora 

📋 LIVRABLES FINAUX

Code Complet

✅ internal/domain/ (15 fichiers)
✅ internal/ports/ (5 fichiers)
✅ internal/services/ (12 fichiers)
✅ internal/adapters/primary/ (10 fichiers)
✅ internal/adapters/secondary/ (15 fichiers)
✅ cmd/ (4 binaires)
✅ migrations/ (10 fichiers up/down)

Tests

✅ tests/unit/ (80%+ coverage)
✅ tests/integration/ (testcontainers)
✅ tests/e2e/ (scénarios complets)
✅ tests/benchmarks/ (performance)

Documentation

✅ docs/ARCHITECTURE.md (C4 + ADR)
✅ docs/RUNBOOK.md (incidents)
✅ docs/API.md (REST endpoints)
✅ docs/RADIUS.md (attributes)
✅ docs/SECURITY.md (practices)
✅ docs/COMPLIANCE.md (legal)
✅ README.md (quick start)

Deployment

✅ docker-compose.yml (dev)
✅ docker-compose.prod.yml (prod stack)
✅ kubernetes/ (manifests complets)
✅ terraform/ (infrastructure as code)
✅ .github/workflows/ (CI/CD)

Tooling

✅ Makefile (build, test, migrate, lint)
✅ .golangci.yml (linter config)
✅ scripts/ (backup, failover, seed)

🎯 OBJECTIF FINAL

Le module généré avec ce prompt V8.0 doit être :

✅ Carrier-Grade - Opérateur télécoms production
✅ Multi-Tenant B2B/B2B2C - Opérateur → Fournisseurs → Revendeurs → Clients
✅ Scalable 100k+ users - Architecture horizontale
✅ Disponibilité 99.95% - HA + DR + monitoring
✅ Conforme légal - RGPD + interception + retention
✅ Complet métier - RADIUS + Billing + IPAM + GIS + CRM + Portail
✅ Sécurisé Zero Trust - MFA + RBAC + Audit + Chiffrement
✅ Observable - Metrics + Logs + Tracing + Alerting
✅ Testable - 80%+ coverage + E2E
✅ Deployable - Docker + K8s + Terraform + CI/CD

RAPPEL CARRIER-GRADE (OBLIGATOIRE)

Le code généré DOIT :

⚠️ Compiler sans erreur

⚠️ Contenir tous les imports nécessaires

⚠️ Ne contenir aucun import inutilisé

⚠️ Utiliser regexp.MustCompile uniquement au niveau package

⚠️ Ne pas utiliser de variables globales mutables

⚠️ Être thread-safe

⚠️ Propager context.Context partout

⚠️ Gérer proprement les erreurs (wrapping avec %w)

⚠️ Avoir des tests qui compilent et passent

⚠️ Passer golangci-lint

⚠️ Être compatible Graceful Shutdown

⚠️ Ne pas contenir de TODO

⚠️ Ne pas contenir de panic()

⚠️ Être prêt pour la production carrier-grade ISP system

Retien bien ceci et garde le en mémoire