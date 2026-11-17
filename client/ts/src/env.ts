// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ValidationError } from "@/errors";

/** Environment variables, such as `process.env`. */
export type Env = Readonly<Record<string, string | undefined>>;

/** The connection parameters that the environment variables give. */
export interface Params {
  host: string;
  port: number;
  apiKey: string;
  secure: boolean;
}

const TRUE = ["1", "true", "yes"];
const FALSE = ["", "0", "false", "no"];

// Values from env files and secrets often carry a trailing newline, and an empty value
// usually means an unset variable in Docker Compose or CI.
const read = (env: Env, name: string): string => env[name]?.trim() ?? "";

/**
 * Loads the connection parameters in the SYNNAX_HOST, SYNNAX_PORT, SYNNAX_API_KEY,
 * and SYNNAX_SECURE environment variables. Values are trimmed, and an empty value
 * counts as unset.
 * @param env - The environment variables, or undefined when the runtime has none.
 * @throws {ValidationError} If SYNNAX_HOST and SYNNAX_API_KEY are not both set, or if
 * SYNNAX_PORT or SYNNAX_SECURE is not valid.
 */
export const loadParams = (env: Env = {}): Params => {
  const host = read(env, "SYNNAX_HOST");
  const apiKey = read(env, "SYNNAX_API_KEY");
  if (host === "" && apiKey === "")
    throw new ValidationError("give a host, or set SYNNAX_HOST and SYNNAX_API_KEY");
  if (host === "") throw new ValidationError("set SYNNAX_HOST with SYNNAX_API_KEY");
  if (apiKey === "") throw new ValidationError("set SYNNAX_API_KEY with SYNNAX_HOST");
  const rawPort = read(env, "SYNNAX_PORT") || "9090";
  const port = /^\d+$/.test(rawPort) ? Number(rawPort) : 0;
  if (port < 1 || port > 65535)
    throw new ValidationError(
      `SYNNAX_PORT must be a port from 1 to 65535, not "${rawPort}"`,
    );
  const rawSecure = read(env, "SYNNAX_SECURE");
  const secure = TRUE.includes(rawSecure.toLowerCase());
  if (!secure && !FALSE.includes(rawSecure.toLowerCase()))
    throw new ValidationError(
      `SYNNAX_SECURE must be true or false, not "${rawSecure}"`,
    );
  return { host, port, apiKey, secure };
};
