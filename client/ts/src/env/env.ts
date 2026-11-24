// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ChildProcess } from "node:child_process";

import { errors, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

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
  oidc?: object;
  secure?: boolean;
  caCert?: string;
}

/** The output of a program that ran to its end. */
export interface Run {
  status: number | null;
  stdout: string;
  stderr: string;
}

/** The access to the operating system that finds Synnax Desktop and CA files. */
export interface System {
  /**
   * Returns the text of the file at path, or null when there is no file. A leading `~`
   * is the home directory.
   */
  readFile: (path: string) => string | null;
  /**
   * Runs the program at path with no arguments. Resolves null if there is none. The
   * signal stops the program.
   */
  run: (path: string, signal?: AbortSignal) => Promise<Run | null>;
  /**
   * True in Deno, where TLS trusts only the store that --cert or DENO_CERT fills, so
   * the SYNNAX_CA_CERT file is not read.
   */
  caCertUnsupported?: boolean;
}

interface Resolved {
  host: string;
  port: number | string;
  apiKey?: string;
  /** Loads the API key from Synnax Desktop, when no other source gives one. */
  loadAPIKey?: (signal?: AbortSignal) => Promise<string>;
  secure: boolean;
  caCert?: string;
}

interface Desktop {
  host: string;
  port: number;
  secure: boolean;
  loadAPIKey: (signal?: AbortSignal) => Promise<string>;
}

const DEFAULT_PORT = 9090;
const TRUE = ["1", "true", "yes"];
const FALSE = ["0", "false", "no"];

const DESKTOP_FILE = "~/.synnax/desktop.json";
const NO_HOST_MSG = "give a host, or set SYNNAX_HOST";
const OLD_DESKTOP_MSG =
  "Synnax Desktop saved its API key in an older format. Update Synnax Desktop and " +
  "open it.";
const NOT_RUNNING_MSG =
  "Synnax Desktop is not running. Open it and try again. To connect to another Core, " +
  "give a host and a port.";
const UNREADABLE_MSG =
  "Cannot read ~/.synnax/desktop.json. Update Synnax Desktop and open it.";
const CA_NEEDS_SECURE_MSG =
  "SYNNAX_CA_CERT requires SYNNAX_SECURE; leave SYNNAX_SECURE out or set it to true";
const HELPER_TIMEOUT = TimeSpan.seconds(30);

const desktopFileZ = z.object({
  host: z.string(),
  port: z.number(),
  secure: z.boolean(),
  api_key_helper: z.string(),
});

const isNotFound = (e: Error | undefined): boolean =>
  e != null && "code" in e && e.code === "ENOENT";

// Deno throws NotCapable when the script has no permission. Its message names the flag.
const isNotCapable = (err: Error): boolean => err.name === "NotCapable";

// Values from env files and secrets often carry a trailing newline, and an empty value
// usually means an unset variable in Docker Compose or CI.
const read = (env: Env, name: string): string => {
  try {
    return env[name]?.trim() ?? "";
  } catch (e) {
    const err = errors.fromUnknown(e);
    if (isNotCapable(err)) return "";
    throw err;
  }
};

const readPort = (env: Env): number | undefined => {
  const raw = read(env, "SYNNAX_PORT");
  if (raw === "") return undefined;
  const port = /^\d+$/.test(raw) ? Number(raw) : 0;
  if (port < 1 || port > 65535)
    throw new ValidationError(
      `SYNNAX_PORT must be a port from 1 to 65535, not "${raw}"`,
    );
  return port;
};

const readSecure = (env: Env): boolean | undefined => {
  const raw = read(env, "SYNNAX_SECURE");
  if (raw === "") return undefined;
  const value = raw.toLowerCase();
  if (TRUE.includes(value)) return true;
  if (FALSE.includes(value)) return false;
  throw new ValidationError(`SYNNAX_SECURE must be true or false, not "${raw}"`);
};

// A missing file or a Deno permission error means no Synnax Desktop that the script can
// see, so the message does not name it.
const readDesktopFile = (system: System): string => {
  let text: string | null;
  try {
    text = system.readFile(DESKTOP_FILE);
  } catch (e) {
    const err = errors.fromUnknown(e);
    if (isNotCapable(err)) throw new ValidationError(NO_HOST_MSG, { cause: err });
    throw new ValidationError(`Cannot read ~/.synnax/desktop.json: ${err.message}`, {
      cause: err,
    });
  }
  if (text == null) throw new ValidationError(NO_HOST_MSG);
  return text;
};

// The user chose this file, so unlike ~/.synnax/desktop.json, a Deno permission error
// is not the same as no file.
const readCA = (system: System | undefined, path: string): string => {
  const cannotRead = (reason: string, cause?: Error): ValidationError =>
    new ValidationError(`Cannot read SYNNAX_CA_CERT file "${path}": ${reason}`, {
      cause,
    });
  if (system == null) throw cannotRead("this runtime has no file access");
  let text: string | null;
  try {
    text = system.readFile(path);
  } catch (e) {
    const err = errors.fromUnknown(e);
    throw cannotRead(err.message, err);
  }
  if (text == null)
    throw new ValidationError(`SYNNAX_CA_CERT names "${path}", which does not exist`);
  if (text.trim() === "")
    throw new ValidationError(`SYNNAX_CA_CERT names "${path}", which is empty`);
  if (!text.includes("-----BEGIN CERTIFICATE-----"))
    throw new ValidationError(`SYNNAX_CA_CERT names "${path}", which is not PEM text`);
  return text;
};

const resolveTLS = (
  params: Given,
  env: Env,
  system: System | undefined,
  desktop: Desktop | undefined,
): Pick<Resolved, "secure" | "caCert"> => {
  if (params.secure != null || params.caCert != null)
    return { secure: params.secure ?? true, caCert: params.caCert };
  const secure = readSecure(env);
  const path = read(env, "SYNNAX_CA_CERT");
  if (path === "") return { secure: secure ?? desktop?.secure ?? false };
  if (secure === false) throw new ValidationError(CA_NEEDS_SECURE_MSG);
  if (system?.caCertUnsupported === true) return { secure: true };
  return { secure: true, caCert: readCA(system, path) };
};

const runHelper = async (
  system: System,
  helper: string,
  signal?: AbortSignal,
): Promise<Run | null> => {
  try {
    return await system.run(helper, signal);
  } catch (e) {
    const err = errors.fromUnknown(e);
    if (!isNotCapable(err)) throw err;
    throw new ValidationError(
      `Cannot run the key helper of Synnax Desktop: ${err.message}`,
      { cause: err },
    );
  }
};

const readKey = async (
  system: System,
  helper: string,
  signal?: AbortSignal,
): Promise<string> => {
  const out = await runHelper(system, helper, signal);
  if (out == null) throw new ValidationError(NOT_RUNNING_MSG);
  if (out.status !== 0)
    throw new ValidationError(
      out.stderr.trim() || "The key helper of Synnax Desktop failed",
    );
  return out.stdout.trim();
};

const loadDesktop = (system: System | undefined): Desktop => {
  if (system == null) throw new ValidationError(NO_HOST_MSG);
  const text = readDesktopFile(system);
  let raw: unknown;
  try {
    raw = JSON.parse(text);
  } catch (e) {
    throw new ValidationError(UNREADABLE_MSG, { cause: e });
  }
  if (typeof raw === "object" && raw != null && "api_key" in raw)
    throw new ValidationError(OLD_DESKTOP_MSG);
  const parsed = desktopFileZ.safeParse(raw);
  if (!parsed.success)
    throw new ValidationError(UNREADABLE_MSG, { cause: parsed.error });
  const { api_key_helper: helper, ...file } = parsed.data;
  return {
    ...file,
    loadAPIKey: async (signal) => await readKey(system, helper, signal),
  };
};

/**
 * Returns the access to the operating system in Node.js, Deno, and Bun, or undefined
 * in a browser. Loads no module until a method runs.
 * @param helperTimeout - How long the key helper of Synnax Desktop may run.
 */
export const nodeSystem = (
  helperTimeout: TimeSpan = HELPER_TIMEOUT,
): System | undefined => {
  const proc = globalThis.process;
  if (proc?.getBuiltinModule == null) return undefined;
  return {
    caCertUnsupported: "Deno" in globalThis,
    readFile: (path) => {
      const paths = proc.getBuiltinModule("node:path");
      const home =
        path === "~" || path.startsWith("~/") || path.startsWith(`~${paths.sep}`);
      const file = home
        ? paths.join(proc.getBuiltinModule("node:os").homedir(), path.slice(1))
        : path;
      try {
        return proc.getBuiltinModule("node:fs").readFileSync(file, "utf8");
      } catch (e) {
        const err = errors.fromUnknown(e);
        if (isNotFound(err)) return null;
        throw err;
      }
    },
    run: async (path, signal) =>
      await new Promise((resolve, reject) => {
        const timeout = AbortSignal.timeout(helperTimeout.milliseconds);
        const stop = signal == null ? timeout : AbortSignal.any([signal, timeout]);
        // older Deno versions give a child that failed to spawn no streams
        const child: ChildProcess = proc
          .getBuiltinModule("node:child_process")
          .spawn(path, { signal: stop });
        let stdout = "";
        let stderr = "";
        child.on("error", (err) => {
          if (isNotFound(err)) return resolve(null);
          if (isNotCapable(err)) return reject(err);
          if (signal?.aborted === true)
            return reject(
              new Error(`Stopped the key helper of Synnax Desktop at ${path}`, {
                cause: err,
              }),
            );
          if (timeout.aborted)
            return reject(
              new ValidationError(
                `The key helper of Synnax Desktop at ${path} did not finish in ` +
                  `${helperTimeout.toString()}`,
              ),
            );
          reject(
            new ValidationError(
              `Cannot run the key helper of Synnax Desktop at ${path}: ` +
                `${err.message}. Update Synnax Desktop and open it.`,
              { cause: err },
            ),
          );
        });
        child.on("close", (status) => resolve({ status, stdout, stderr }));
        child.stdout?.setEncoding("utf8").on("data", (d: string) => (stdout += d));
        child.stderr?.setEncoding("utf8").on("data", (d: string) => (stderr += d));
      }),
  };
};

/**
 * Fills each connection parameter from the params, then its environment variable, then
 * Synnax Desktop when neither gives a host, then its default (port 9090, secure false).
 * The API key comes from SYNNAX_API_KEY or Synnax Desktop only when the params give no
 * credentials, token, or OpenID Connect code. The key of Synnax Desktop comes as
 * loadAPIKey, which runs its key helper and rejects with a ValidationError if the
 * helper is gone, fails, or does not finish in time. Secure and the CA certificate come
 * as a pair from the first source that sets either one: the params (secure, caCert),
 * then the variables (SYNNAX_SECURE, and SYNNAX_CA_CERT as the path of a PEM file),
 * then Synnax Desktop. A CA certificate with no secure turns secure on. When
 * system.caCertUnsupported is true, SYNNAX_CA_CERT turns secure on but its file is not
 * read. Reads only the variables of missing fields. Variable values are trimmed, and an
 * empty value is unset. In Deno, a variable the script has no permission to read is
 * unset.
 * @param system - Finds Synnax Desktop and reads the SYNNAX_CA_CERT file. Undefined in
 * a runtime that cannot.
 * @throws {ValidationError} If no host is given, set, or saved by Synnax Desktop, if
 * SYNNAX_PORT or SYNNAX_SECURE is not valid, if SYNNAX_CA_CERT is set with
 * SYNNAX_SECURE false, if the SYNNAX_CA_CERT file is missing, empty, not PEM text, or
 * cannot be read, if ~/.synnax/desktop.json cannot be read, or if Synnax Desktop is
 * too old. In Deno, a missing permission to read ~/.synnax/desktop.json is the same as
 * no file.
 */
export const resolveParams = <P extends Given>(
  params: P,
  env: Env = {},
  system?: System,
): Omit<P, keyof Resolved> & Resolved => {
  const given =
    params.host != null && params.host !== "" ? params.host : read(env, "SYNNAX_HOST");
  const desktop = given === "" ? loadDesktop(system) : undefined;
  const { apiKey, username, password, token, oidc } = params;
  const credentials = apiKey ?? username ?? password ?? token ?? oidc;
  let key = apiKey;
  let loadAPIKey: (() => Promise<string>) | undefined;
  if (credentials == null) {
    const envKey = read(env, "SYNNAX_API_KEY");
    if (envKey !== "") key = envKey;
    else loadAPIKey = desktop?.loadAPIKey;
  }
  return {
    ...params,
    host: desktop?.host ?? given,
    port: params.port ?? readPort(env) ?? desktop?.port ?? DEFAULT_PORT,
    apiKey: key,
    loadAPIKey,
    ...resolveTLS(params, env, system, desktop),
  };
};
