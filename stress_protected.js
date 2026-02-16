import http from 'k6/http';
import { check, sleep } from 'k6';

// CONFIGURATION DU TEST
export const options = {
  // On simule une montée en charge agressive car cette route est très légère
  stages: [
    { duration: '5s', target: 50 },   // Montée rapide à 50 utilisateurs
    { duration: '10s', target: 200 }, // Pointe à 200 utilisateurs simultanés
    { duration: '10s', target: 500 }, // Stress test : 500 utilisateurs !
    { duration: '5s', target: 0 },    // Redescente
  ],
  thresholds: {
    // On exige que 95% des requêtes soient traitées en moins de 50ms
    http_req_duration: ['p(95)<50'], 
    // Zéro erreur tolérée
    http_req_failed: ['rate<0.01'],
  },
};

// 1. SETUP : S'exécute une seule fois au début pour récupérer un Token valide
export function setup() {
  const url = 'http://localhost:8080/v1/login';
  const payload = JSON.stringify({
    username: 'yvan',
    password: 'password123',
    tenant_id: 'tenant-1',
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(url, payload, params);
  
  if (res.status !== 200) {
    throw new Error(`Login failed during setup: ${res.status} ${res.body}`);
  }

  const token = res.json('access_token');
  console.log(`Token récupéré : ${token.substring(0, 15)}...`);
  return token; // Ce token sera passé à la fonction default
}

// 2. SCÉNARIO : Les utilisateurs bombardent la route protégée avec le token
export default function (accessToken) {
  const url = 'http://localhost:8080/v1/user/me';
  
  const params = {
    headers: {
      'Authorization': `Bearer ${accessToken}`,
      'Content-Type': 'application/json',
    },
  };

  const res = http.get(url, params);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'access granted': (r) => r.json('message') === 'Accès autorisé à la zone sécurisée',
  });

  // Pause infime pour simuler un trafic intense (API Gateway)
  sleep(0.01);
}