// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";

import { ValidationError } from "@/errors";

/** Environment variables, such as `process.env`. */
export type Env = Readonly<Record<string, string | undefined>>;

/** The client parameters that the environment variables can fill. */
export interface Given {
  host?: string;
  port?: number | string;
  apiKey?: string;
  username?: string;
  password?: string;
  token?: string;
  secure?: boolean;
  caCert?: string;
}

interface Resolved {
  host: string;
  port: number | string;
  apiKey?: string;
  secure: boolean;
}

const DEFAULT_PORT = 9090;
const TRUE = ["1", "true", "yes"];
const FALSE = ["", "0", "false", "no"];

// Values from env files and secrets often carry a trailing newline, and an empty value
// usually means an unset variable in Docker Compose or CI.
const read = (env: Env, name: string): string => {
  try {
    return env[name]?.trim() ?? "";
  } catch (e) {
    const err = errors.fromUnknown(e);
    // Deno throws NotCapable when the script has no permission to read the variable.
    if (err.name === "NotCapable") return "";
    throw err;
  }
};

const readPort = (env: Env): number => {
  const raw = read(env, "SYNNAX_PORT");
  if (raw === "") return DEFAULT_PORT;
  const port = /^\d+$/.test(raw) ? Number(raw) : 0;
  if (port < 1 || port > 65535)
    throw new ValidationError(
      `SYNNAX_PORT must be a port from 1 to 65535, not "${raw}"`,
    );
  return port;
};

const readSecure = (env: Env): boolean => {
  const raw = read(env, "SYNNAX_SECURE");
  const secure = TRUE.includes(raw.toLowerCase());
  if (!secure && !FALSE.includes(raw.toLowerCase()))
    throw new ValidationError(`SYNNAX_SECURE must be true or false, not "${raw}"`);
  return secure;
};

/**
 * Fills each connection parameter from the params, then its environment variable,
 * then its default (port 9090, secure false). SYNNAX_API_KEY fills the API key only
 * when the params give no credentials or token. Secure and the CA certificate come as a
 * pair: when the params give either one, SYNNAX_SECURE is not read, and a CA
 * certificate with no secure turns secure on. Reads only the variables of missing
 * fields. Variable values are trimmed, and an empty value is unset. In Deno, a variable
 * the script has no permission to read is unset.
 * @throws {ValidationError} If neither the params nor SYNNAX_HOST give a host, or if
 * SYNNAX_PORT or SYNNAX_SECURE is not valid.
 */
export const resolveParams = <P extends Given>(
  params: P,
  env: Env = {},
): Omit<P, keyof Resolved> & Resolved => {
  const host =
    params.host != null && params.host !== "" ? params.host : read(env, "SYNNAX_HOST");
  if (host === "") throw new ValidationError("give a host, or set SYNNAX_HOST");
  const { apiKey, username, password, token } = params;
  const credentials = apiKey ?? username ?? password ?? token;
  const envKey = credentials == null ? read(env, "SYNNAX_API_KEY") : "";
  return {
    ...params,
    host,
    port: params.port ?? readPort(env),
    apiKey: envKey === "" ? apiKey : envKey,
    secure:
      params.secure != null || params.caCert != null
        ? (params.secure ?? true)
        : readSecure(env),
  };
};
