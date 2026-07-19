const requiredVariables = [
  'MAESTRO_E2E_API_URL',
  'MAESTRO_E2E_USERNAME',
  'MAESTRO_E2E_EMAIL',
  'MAESTRO_E2E_PASSWORD',
];

for (const name of requiredVariables) {
  if (!process.env[name]?.trim()) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
}

const apiURL = process.env.MAESTRO_E2E_API_URL.replace(/\/+$/, '');
const username = process.env.MAESTRO_E2E_USERNAME.trim();

const response = await fetch(`${apiURL}/api/auth/register`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    username,
    email: process.env.MAESTRO_E2E_EMAIL.trim(),
    password: process.env.MAESTRO_E2E_PASSWORD,
  }),
});
const envelope = await response.json().catch(() => null);

if (!response.ok || !envelope || envelope.code !== 0) {
  const message = envelope?.message || `HTTP ${response.status}`;
  throw new Error(`/api/auth/register failed: ${message}`);
}

if (!envelope.data?.access_token || envelope.data.username !== username) {
  throw new Error('registration did not return the expected disposable user session');
}

console.log(`Prepared disposable user fixture for ${username}.`);
