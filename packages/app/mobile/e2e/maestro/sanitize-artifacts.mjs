#!/usr/bin/env node

import {
  existsSync,
  readdirSync,
  readFileSync,
  unlinkSync,
  writeFileSync,
} from 'node:fs';
import { extname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const secretEnvironmentVariables = [
  'MAESTRO_E2E_PASSWORD',
  'MAESTRO_E2E_OLD_PASSWORD',
  'MAESTRO_E2E_WRONG_PASSWORD',
  'MAESTRO_E2E_NEW_PASSWORD',
];
const textExtensions = new Set(['.json', '.log', '.txt', '.xml', '.yaml', '.yml']);

function sanitizeDirectory(directory, secrets) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);

    if (entry.isDirectory()) {
      sanitizeDirectory(path, secrets);
      continue;
    }

    if (entry.name.startsWith('commands-') && entry.name.endsWith('.json')) {
      unlinkSync(path);
      continue;
    }

    if (!textExtensions.has(extname(entry.name))) {
      continue;
    }

    const original = readFileSync(path, 'utf8');
    const sanitized = secrets.reduce(
      (content, secret) => content.split(secret).join('<redacted>'),
      original,
    );

    if (sanitized !== original) {
      writeFileSync(path, sanitized, 'utf8');
    }
  }
}

export function sanitizeArtifacts(artifactRoot, environment = process.env) {
  if (!existsSync(artifactRoot)) {
    return;
  }

  const secrets = secretEnvironmentVariables
    .map((name) => environment[name])
    .filter((value) => typeof value === 'string' && value.length > 0);

  sanitizeDirectory(artifactRoot, secrets);
}

const isEntryPoint = process.argv[1]
  && resolve(process.argv[1]) === fileURLToPath(import.meta.url);

if (isEntryPoint) {
  const artifactRoot = process.argv[2];

  if (!artifactRoot) {
    throw new Error('Usage: sanitize-artifacts.mjs <artifact-directory>');
  }

  sanitizeArtifacts(artifactRoot);
}
