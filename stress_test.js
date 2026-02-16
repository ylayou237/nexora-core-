import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '5s', target: 20 },  // Montée en charge rapide
    { duration: '20s', target: 50 }, // Plateau (ajuste selon ton CPU)
    { duration: '5s', target: 0 },  // Descente
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'], // On veut que 95% des logins soient < 200ms
    http_req_failed: ['rate<0.01'],   // Moins de 1% d'erreurs
  },
};

export default function () {
  const url = 'http://localhost:8080/v1/login';
  
  // Le corps de la requête doit matcher ton struct LoginRequest
  const payload = JSON.stringify({
    username: 'yvan',
    password: 'password123',
    tenant_id: 'tenant-default',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(url, payload, params);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has token pair': (r) => {
      const body = r.json();
      return body.access_token !== undefined && body.refresh_token !== undefined;
    },
  });

  // Avec Bcrypt, ne pas mettre un sleep trop court pour laisser le CPU respirer
  sleep(0.1);
}