// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { generateKeyPairSync, verify } from "node:crypto";

import { type APIContext } from "astro";
import { expect } from "vitest";

import { type Portal } from "@/portal";
import {
  type Activation,
  activation,
  type License,
  license,
  type Organization,
  organization,
} from "@/server/db/schema";
import { type Memory } from "@/server/db/testutil";
import { type Directory, memory as directory, type Records } from "@/server/directory";
import { local } from "@/server/license/sign";
import { NOW } from "@/server/license/testutil";
import { type Mailer, memory as mailer, type Message } from "@/server/mail";
import { resolve } from "@/server/session";

export const STAFF_ORG_ID = "org_staff";
export const CRON_SECRET = "cron-secret";
export const WEBHOOK_SECRET = `whsec_${Buffer.from("secret").toString("base64")}`;

const { privateKey, publicKey } = generateKeyPairSync("ml-dsa-44");

/** Harness is a portal on an in-memory store, directory, and mailer. */
export interface Harness {
  portal: Portal;
  store: Memory;
  directory: Directory & Records;
  mail: Mailer & { sent: Message[] };
  /** signIn makes later sessions resolve as the user, or as nobody for null. */
  signIn: (userID: string | null) => void;
}

/** createHarness wires a portal whose clock reads {@link NOW}. */
export const createHarness = (store: Memory): Harness => {
  const dir = directory();
  const mail = mailer();
  let userID: string | null = null;
  return {
    store,
    directory: dir,
    mail,
    signIn: (id) => {
      userID = id;
    },
    portal: {
      store,
      signer: local(privateKey, "test"),
      mail,
      directory: dir,
      staffOrgID: STAFF_ORG_ID,
      cronSecret: CRON_SECRET,
      webhookSecret: WEBHOOK_SECRET,
      session: async () => await resolve(dir, userID, STAFF_ORG_ID),
      now: () => NOW,
    },
  };
};

/**
 * readKey verifies a license key against the harness signing key and returns its
 * claims.
 */
export const readKey = (key: string): Record<string, unknown> => {
  const [header, payload, signature] = key.split(".");
  expect(
    verify(
      null,
      Buffer.from(`${header}.${payload}`),
      publicKey,
      Buffer.from(signature, "base64url"),
    ),
  ).toBe(true);
  return JSON.parse(Buffer.from(payload, "base64url").toString()) as Record<
    string,
    unknown
  >;
};

export interface ContextArgs {
  params?: Record<string, string>;
  method?: string;
  path?: string;
  /** body posts as JSON unless it is FormData. */
  body?: unknown;
  headers?: Record<string, string>;
  cookies?: Record<string, string>;
}

/** Cookies is the cookie jar a {@link createAPIContext} context reads and writes. */
export type Cookies = Map<string, string>;

/** createAPIContext builds the request context a route or page handler receives. */
export const createAPIContext = (
  portal: Portal,
  {
    params = {},
    method = "POST",
    path = "/",
    body,
    headers = {},
    cookies = {},
  }: ContextArgs = {},
): APIContext & { jar: Cookies } => {
  const url = new URL(path, "https://portal.synnaxlabs.com");
  const init: RequestInit = { method, headers: { ...headers } };
  if (body instanceof FormData) init.body = body;
  else if (body !== undefined) {
    init.body = JSON.stringify(body);
    init.headers = { "content-type": "application/json", ...headers };
  }
  const jar: Cookies = new Map(Object.entries(cookies));
  const context = {
    params,
    url,
    request: new Request(url, init),
    locals: { portal },
    jar,
    cookies: {
      get: (name: string) => {
        const value = jar.get(name);
        return value == null ? undefined : { value };
      },
      set: (name: string, value: string) => void jar.set(name, value),
    },
    redirect: (location: string, status = 302) =>
      new Response(null, { status, headers: { location } }),
  };
  return context as unknown as APIContext & { jar: Cookies };
};

/** body reads a JSON response body. */
export const body = async <T = Record<string, unknown>>(res: Response): Promise<T> =>
  (await res.json()) as T;

export const createOrganization = async (
  store: Memory,
  values: Partial<Organization> & Pick<Organization, "kind">,
): Promise<Organization> => {
  const [row] = await store.query
    .insert(organization)
    .values({ name: values.kind === "team" ? "Acme" : "Ada Lovelace", ...values })
    .returning();
  return row;
};

export const createLicense = async (
  store: Memory,
  values: Partial<License> & Pick<License, "organization">,
): Promise<License> => {
  const [row] = await store.query
    .insert(license)
    .values({
      edition: "enterprise",
      term: "subscription",
      nodes: 2,
      channels: 0,
      expiresAt: new Date("2027-03-01T00:00:00Z"),
      label: "Test rig",
      issuedBy: "user_staff",
      issuedAt: new Date("2026-09-01T00:00:00Z"),
      ...values,
    })
    .returning();
  return row;
};

export const createActivation = async (
  store: Memory,
  values: Partial<Activation> & Pick<Activation, "license" | "fingerprint">,
): Promise<Activation> => {
  const [row] = await store.query
    .insert(activation)
    .values({ firstSeen: NOW, lastSeen: NOW, ...values })
    .returning();
  return row;
};
