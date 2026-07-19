import {
  existsSync,
  mkdtempSync,
  mkdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';

import { sanitizeArtifacts } from './sanitize-artifacts.mjs';

const artifactRoots = [];

afterEach(() => {
  for (const artifactRoot of artifactRoots.splice(0)) {
    rmSync(artifactRoot, { force: true, recursive: true });
  }
});

describe('sanitizeArtifacts', () => {
  it('deletes expanded commands and redacts passwords from text artifacts', () => {
    const artifactRoot = mkdtempSync(join(tmpdir(), 'cove-maestro-artifacts-'));
    artifactRoots.push(artifactRoot);

    const evidenceDirectory = join(artifactRoot, 'evidence');
    const debugDirectory = join(artifactRoot, 'maestro-debug');
    mkdirSync(evidenceDirectory);
    mkdirSync(debugDirectory);

    const commandPath = join(evidenceDirectory, 'commands-flow.json');
    const debugLogPath = join(debugDirectory, 'maestro.log');
    writeFileSync(commandPath, '{"password":"synthetic-secret"}', 'utf8');
    writeFileSync(debugLogPath, 'Inputting text: synthetic-secret', 'utf8');

    sanitizeArtifacts(artifactRoot, {
      MAESTRO_E2E_PASSWORD: 'synthetic-secret',
    });

    expect(existsSync(commandPath)).toBe(false);
    expect(readFileSync(debugLogPath, 'utf8')).toBe('Inputting text: <redacted>');
  });
});
