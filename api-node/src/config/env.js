import { existsSync } from 'node:fs';

if (existsSync('.env')) {
  process.loadEnvFile('.env');
}

export function getConfig(environment = process.env) {
  const port = Number.parseInt(environment.PORT ?? '3001', 10);

  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new TypeError('PORT debe ser un número entero entre 1 y 65535.');
  }

  return { port };
}

export const config = getConfig();
