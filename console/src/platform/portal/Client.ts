// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

export type RenewResult =
  { variant: "renewed"; key: string } | { variant: "unlinked"; message: string };

/** The code the hub puts on a refusal for a machine it no longer knows. */
const UNLINKED_CODE = "unlinked";

const UNLINK_TIMEOUT = TimeSpan.seconds(5);
const RECEIVE_TIMEOUT = TimeSpan.seconds(10);

const refusalZ = z.object({ error: z.string(), code: z.string().optional() });
const keyZ = z.object({ key: z.string() });

interface Refusal {
  message: string;
  code?: string;
}

const readRefusal = async (res: Response): Promise<Refusal> => {
  try {
    const body = refusalZ.safeParse(await res.json());
    if (body.success) return { message: body.data.error, code: body.data.code };
  } catch {
    // A body that is not JSON carries no message.
  }
  return { message: res.statusText };
};

/** Joins the hashes of a fingerprint the way the portal reads them. */
export const joinFingerprint = (fingerprint: string[]): string =>
  fingerprint.join(", ");

export interface Config {
  /** The portal's origin. */
  url: string;
  fetch: typeof fetch;
}

export interface ActivateParams {
  fingerprint: string[];
  /** The code the Console collects the license key with. */
  handoff: string;
}

export interface LoginParams {
  state: string;
  fingerprint: string[];
  /** The name this machine shows in the account. */
  name: string;
  version?: string;
}

/** The pages and routes of the portal, the hub that issues license keys. */
export class Client {
  /** The portal overview of the logged-in account. */
  readonly accountURL: string;
  private readonly url: string;
  private readonly fetch: typeof fetch;

  constructor({ url, fetch }: Config) {
    this.url = url;
    this.fetch = fetch;
    this.accountURL = `${url}/`;
  }

  /** The portal page that issues a license key for a machine and hands it back. */
  activateURL({ fingerprint, handoff }: ActivateParams): string {
    const url = new URL(`${this.url}/licenses/activate`);
    url.searchParams.set("fingerprint", fingerprint.join(","));
    url.searchParams.set("handoff", handoff);
    return url.toString();
  }

  /** The page that links a Synnax Desktop machine to an account. */
  loginURL({ state, fingerprint, name, version }: LoginParams): string {
    const url = new URL(`${this.url}/desktop/login`);
    url.searchParams.set("state", state);
    url.searchParams.set("fp", joinFingerprint(fingerprint));
    url.searchParams.set("name", name);
    if (version != null) url.searchParams.set("v", version);
    return url.toString();
  }

  /**
   * Asks the hub for a fresh license key for the machine the secret belongs to.
   * Unlinked means the hub no longer knows the machine, so the secret is spent. Only
   * the hub's own unlinked answer counts: a proxy can refuse with the same status.
   * @throws {Error} if the hub cannot be reached or refuses for another reason.
   */
  async renew(secret: string): Promise<RenewResult> {
    const res = await this.fetch(`${this.url}/api/desktop/renew`, {
      method: "POST",
      headers: { authorization: `Bearer ${secret}` },
    });
    if (res.ok) {
      const { key } = keyZ.parse(await res.json());
      return { variant: "renewed", key };
    }
    const { message, code } = await readRefusal(res);
    if ((res.status === 401 || res.status === 403) && code === UNLINKED_CODE)
      return { variant: "unlinked", message };
    throw new Error(`Could not renew the license: ${message}`);
  }

  /**
   * Asks the hub to log out the machine the renewal secret belongs to. The machine
   * leaves its account, and the secret stops working.
   * @throws {Error} if the hub refuses, or does not answer within five seconds.
   */
  async unlink(secret: string): Promise<void> {
    const res = await this.fetch(`${this.url}/api/desktop/unlink`, {
      method: "POST",
      headers: { authorization: `Bearer ${secret}` },
      signal: AbortSignal.timeout(UNLINK_TIMEOUT.milliseconds),
    });
    if (res.ok) return;
    const { message } = await readRefusal(res);
    throw new Error(`Could not log out: ${message}`);
  }

  /**
   * Asks the hub for the license key it holds for a handoff code. The hub gives the
   * key out once.
   * @returns the key, or null while the hub holds none for the code.
   * @throws {Error} if the hub refuses.
   * @throws {TypeError} if the hub cannot be reached, or does not answer within ten
   * seconds.
   */
  async receive(code: string, signal?: AbortSignal): Promise<string | null> {
    // A timer, not AbortSignal.timeout, so fake timers can drive the timeout in specs.
    const timeout = new AbortController();
    const timer = setTimeout(() => timeout.abort(), RECEIVE_TIMEOUT.milliseconds);
    let res: Response;
    try {
      res = await this.fetch(`${this.url}/api/handoff`, {
        method: "POST",
        headers: { authorization: `Bearer ${code}` },
        signal:
          signal == null ? timeout.signal : AbortSignal.any([signal, timeout.signal]),
      });
    } catch (e) {
      if (timeout.signal.aborted && signal?.aborted !== true)
        throw new TypeError("The hub did not answer in time", { cause: e });
      throw errors.fromUnknown(e);
    } finally {
      clearTimeout(timer);
    }
    if (res.status === 204) return null;
    if (res.ok) return keyZ.parse(await res.json()).key;
    const { message } = await readRefusal(res);
    throw new Error(`Could not get the license key: ${message}`);
  }
}
