C'est un structure **parfaite**. Elle respecte scrupuleusement l'Architecture Hexagonale (Ports & Adapters) et sépare clairement le frontend (`web`) du backend (`cmd`, `internal`).

Voici ton **`README.md` complet et mis à jour** pour refléter exactement cette structure de dossiers. Copie-colle ce contenu à la racine de ton projet.

```markdown
# 🚀 Nexora-Core V8.0

![CI Status](https://github.com/yvan/nexora-core/actions/workflows/ci.yml/badge.svg)
![Go Version](https://img.shields.io/github/go-mod/go-version/yvan/nexora-core)
![Architecture](https://img.shields.io/badge/architecture-Hexagonal-blue)
![License](https://img.shields.io/badge/license-Proprietary-red)

**Plateforme ISP/WISP Carrier-Grade multi-tenant.**

Nexora est le moteur central pour la gestion des fournisseurs d'accès internet, intégrant l'authentification RADIUS haute performance, la facturation, le GIS et un portail captif dynamique.

## 🏗️ Architecture Hexagonale

Ce projet suit strictement le pattern **Ports & Adapters**. Le code métier est isolé de la technologie.

### 🗺️ Cartographie du Code

```text
nexora-core/
├── cmd/                        # Points d'entrée (Binaires)
│   ├── api/                    # Serveur REST API (Gin)
│   ├── radius-server/          # Serveur UDP RADIUS
│   ├── worker/                 # Tâches de fond (Cron jobs)
│   └── accounting-consumer/    # Consommateur NATS (Logs de connexion)
├── internal/
│   ├── core/                   # 🟢 LE COEUR (Pur Go, zéro dépendance externe)
│   │   ├── domain/             # Entités (User, Invoice, Session)
│   │   ├── ports/              # Interfaces (Ce que le coeur attend)
│   │   └── services/           # Logique métier (Use Cases)
│   └── adapters/               # 🔴 LES ADAPTATEURS (Implémentations techniques)
│       ├── primary/            # Entrants (Ce qui pilote l'app)
│       │   ├── web/            # API Handlers, DTOs, Middlewares
│       │   ├── radius/         # Serveur RADIUS Drivers
│       │   └── captive/        # Templates HTML Portail Captif
│       └── secondary/          # Sortants (Infrastructure)
│           ├── postgres/       # Base de données
│           ├── redis/          # Cache & Sessions
│           ├── payment/        # Stripe/PayPal
│           ├── notification/   # SMS/Email (SMTP, Twilio)
│           ├── gis/            # Cartographie & Éligibilité
│           └── queue/          # NATS JetStream
├── deployments/                # Kubernetes (K8s) & Terraform
├── web/                        # Frontend (React/Vue)
└── tests/                      # Tests Unitaires, Intégration, E2E & Benchmarks

```

## 📦 Modules & Fonctionnalités

* **RADIUS Server** : Authentification haute performance (RFC 2865, 2866, 3576).
* **Billing & Accounting** : Gestion des factures, wallets et réconciliation via `accounting-consumer`.
* **Portail Captif** : Gestion des sessions Hotspot avec templates dynamiques.
* **GIS** : Gestion de l'éligibilité réseau (Fibre/Radio) et couverture.
* **Notifications** : Système d'alerte multicanal.

## 🚀 Démarrage Rapide (Développement)

### 1. Pré-requis

* Go 1.22+
* Docker & Docker Compose
* Make
* Node.js (pour le dossier `web/`)

### 2. Installation

```bash
# Cloner le repo
git clone [https://github.com/yvan/nexora-core](https://github.com/yvan/nexora-core)
cd nexora-core

# Installer les dépendances Go
go mod download

# Configurer l'environnement
cp .env.example .env
# (Modifiez .env selon vos besoins)

```

### 3. Lancer l'Infrastructure

Démarre PostgreSQL, Redis, et NATS via Docker :

```bash
make docker-dev

```

### 4. Lancer les Services (Backend)

Vous pouvez lancer les services individuellement :

```bash
# Lancer l'API Principale
go run cmd/api/main.go

# Lancer le Serveur RADIUS
go run cmd/radius-server/main.go

# Lancer le Worker (Tâches de fond)
go run cmd/worker/main.go

```

### 5. Lancer le Frontend

```bash
cd web
npm install
npm run dev

```

## 🧪 Tests & Qualité

Nous utilisons une pyramide de tests complète.

| Type | Commande | Description |
| --- | --- | --- |
| **Unitaires** | `make test-unit` | Teste `internal/core` (Domain/Services) |
| **Intégration** | `make test-integration` | Teste `internal/adapters` avec DB Dockerisée |
| **E2E** | `make test-e2e` | Teste les parcours complets API + Radius |
| **Benchmark** | `make benchmark` | Tests de charge RADIUS |
| **Lint** | `make lint` | Analyse statique (GolangCI-Lint) |

## 🚢 Déploiement

Les configurations pour la production se trouvent dans le dossier `deployments/` :

* **Kubernetes** : Manifests pour cluster K8s (`deployments/kubernetes`).
* **Terraform** : Infrastructure as Code (`deployments/terraform`).
* **Monitoring** : Configuration Prometheus/Grafana (`deployments/kubernetes/monitoring`).

## 📚 Documentation

* [Architecture détaillée](https://www.google.com/search?q=docs/ARCHITECTURE.md)
* [Guide de contribution](https://www.google.com/search?q=docs/CONTRIBUTING.md)
* [API Swagger](https://www.google.com/search?q=docs/API.md)

## 📄 License

Propriétaire - Tous droits réservés © 2026 Nexora Inc.

```

```