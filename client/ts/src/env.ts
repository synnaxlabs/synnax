// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import { errors, url } from "@synnaxlabs/x";
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
  /** Runs the program at path with no arguments. Returns null if there is none. */
  run: (path: string) => Run | null;
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
  secure: boolean;
  caCert?: string;
}

interface Desktop {
  host: string;
  port: number;
  secure: boolean;
  apiKey: () => string;
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
const HELPER_TIMEOUT_MS = 30_000;

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

const runHelper = (system: System, helper: string): Run | null => {
  try {
    return system.run(helper);
  } catch (e) {
    const err = errors.fromUnknown(e);
    if (!isNotCapable(err)) throw err;
    throw new ValidationError(
      `Cannot run the key helper of Synnax Desktop: ${err.message}`,
      { cause: err },
    );
  }
};

const readKey = (system: System, helper: string, address: url.URL): string => {
  const out = runHelper(system, helper);
  // The helper of an AppImage goes away with its mount when Synnax Desktop closes.
  if (out == null) throw new Unreachable({ message: NOT_RUNNING_MSG, url: address });
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
  const address = new url.URL({ host: file.host, port: file.port });
  return { ...file, apiKey: () => readKey(system, helper, address) };
};

/**
 * Returns the access to the operating system in Node.js, Deno, and Bun, or undefined
 * in a browser. Loads no module until a method runs.
 */
export const nodeSystem = (): System | undefined => {
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
    run: (path) => {
      const out = proc
        .getBuiltinModule("node:child_process")
        .spawnSync(path, { encoding: "utf8", timeout: HELPER_TIMEOUT_MS });
      if (isNotFound(out.error)) return null;
      if (out.error != null)
        throw new ValidationError(
          `Cannot run the key helper of Synnax Desktop at ${path}: ` +
            `${out.error.message}. Update Synnax Desktop and open it.`,
          { cause: out.error },
        );
      // Deno returns an empty result when the script has no permission to run.
      if (out.status == null && out.signal == null)
        throw new ValidationError(
          `Cannot run ${path}. In Deno, run the script with the --allow-run flag.`,
        );
      return { status: out.status, stdout: out.stdout, stderr: out.stderr };
    },
  };
};

/**
 * Fills each connection parameter from the params, then its environment variable,
 * then Synnax Desktop when neither gives a host, then its default (port 9090, secure
 * false). The API key comes from SYNNAX_API_KEY or Synnax Desktop only when the params
 * give no credentials or token. Secure and the CA certificate come as a pair from the
 * first source that sets either one: the params (secure, caCert), then the variables
 * (SYNNAX_SECURE, and SYNNAX_CA_CERT as the path of a PEM file), then Synnax Desktop. A
 * CA certificate with no secure turns secure on. When system.caCertUnsupported is
 * true, SYNNAX_CA_CERT turns secure on but its file is not read. Reads only the
 * variables of missing fields. Variable values are trimmed, and an empty value is
 * unset. In Deno, a variable the script has no permission to read is unset.
 * @param system - Finds Synnax Desktop and reads the SYNNAX_CA_CERT file. Undefined in
 * a runtime that cannot.
 * @throws {ValidationError} If no host is given, set, or saved by Synnax Desktop, if
 * SYNNAX_PORT or SYNNAX_SECURE is not valid, if SYNNAX_CA_CERT is set with
 * SYNNAX_SECURE false, if the SYNNAX_CA_CERT file is missing, empty, not PEM text, or
 * cannot be read, if ~/.synnax/desktop.json cannot be read, if Synnax Desktop is too
 * old, or if its key helper fails. In Deno, a missing permission to read
 * ~/.synnax/desktop.json is the same as no file.
 * @throws {Unreachable} If the key helper of Synnax Desktop is gone, as for a closed
 * AppImage.
 */
export const resolveParams = <P extends Given>(
  params: P,
  env: Env = {},
  system?: System,
): Omit<P, keyof Resolved> & Resolved => {
  const given =
    params.host != null && params.host !== "" ? params.host : read(env, "SYNNAX_HOST");
  const desktop = given === "" ? loadDesktop(system) : undefined;
  const { apiKey, username, password, token } = params;
  const credentials = apiKey ?? username ?? password ?? token;
  let key = apiKey;
  if (credentials == null) {
    const envKey = read(env, "SYNNAX_API_KEY");
    key = envKey !== "" ? envKey : desktop?.apiKey();
  }
  return {
    ...params,
    host: desktop?.host ?? given,
    port: params.port ?? readPort(env) ?? desktop?.port ?? DEFAULT_PORT,
    apiKey: key,
    ...resolveTLS(params, env, system, desktop),
  };
};
