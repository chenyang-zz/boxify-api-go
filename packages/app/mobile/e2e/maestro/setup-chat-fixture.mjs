const requiredVariables = [
  'MAESTRO_E2E_API_URL',
  'MAESTRO_E2E_LLM_BASE_URL',
  'MAESTRO_E2E_USERNAME',
  'MAESTRO_E2E_PASSWORD',
];

for (const name of requiredVariables) {
  if (!process.env[name]?.trim()) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
}

const apiURL = process.env.MAESTRO_E2E_API_URL.replace(/\/+$/, '');
const username = process.env.MAESTRO_E2E_USERNAME.trim();

async function request(path, { token, body } = {}) {
  const response = await fetch(`${apiURL}${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
  });
  const envelope = await response.json().catch(() => null);
  if (!response.ok || !envelope || envelope.code !== 0) {
    const message = envelope?.message || `HTTP ${response.status}`;
    throw new Error(`${path} failed: ${message}`);
  }
  return envelope.data;
}

const registered = await request('/api/auth/register', {
  body: {
    username,
    email: `${username}@example.test`,
    password: process.env.MAESTRO_E2E_PASSWORD,
  },
});
if (!registered?.access_token) {
  throw new Error('registration did not return an access token');
}

const model = await request('/api/model-configs/', {
  token: registered.access_token,
  body: {
    type: 'chat',
    provider: 'openai',
    name: `Local deterministic chat ${username}`,
    model_name: 'cove-e2e-chat',
    api_key: 'cove-e2e-local-key',
    base_url: process.env.MAESTRO_E2E_LLM_BASE_URL.replace(/\/+$/, ''),
    is_default: true,
  },
});
if (!model?.id || !model.is_default) {
  throw new Error('model configuration was not persisted as the default');
}

console.log(`Prepared disposable chat fixture for ${username}.`);
