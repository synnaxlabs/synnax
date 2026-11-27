// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Local auth servers for jsdom specs. A browser spec cannot start one, so it stubs
// fetch instead.

import { createServer, type RequestListener } from "node:http";
import type { AddressInfo } from "node:net";

import { auth, type resource, type Synnax } from "@synnaxlabs/client";
import { onTestFinished } from "vitest";

/** The subject of every session a local auth server opens. */
export const LOCAL_SUBJECT: resource.ID = {
  type: "user",
  key: "1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f",
};

const SESSION = {
  key: "8b7c3e52-6f0a-4c1d-9e2b-3a4f5d6e7f80",
  subject: LOCAL_SUBJECT,
  credential: "2d3e4f5a-6b7c-4d8e-9f0a-1b2c3d4e5f60",
  lastActive: 0,
};

type LocalAddress = auth.TokenlessParams & { port: number };

// the server closes when the test finishes
const listen = async (listener: RequestListener): Promise<LocalAddress> => {
  const server = createServer(listener);
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  onTestFinished(
    async () =>
      await new Promise<void>((resolve) => {
        server.close(() => resolve());
        // an idle keep-alive connection would hold the close open
        server.closeAllConnections();
      }),
  );
  return { host: "127.0.0.1", port: (server.address() as AddressInfo).port };
};

/** What a local auth server holds and how it answers. */
export interface ServeOIDCParams {
  methods?: string[];
  providers?: auth.OIDCLogin[];
  links?: auth.oidc.Link[];
  /** The type of the validation error that every link is refused with. */
  linkRefusal?: string;
}

export interface OIDCServer {
  address: LocalAddress;
  /** The links the server holds. */
  links: auth.oidc.Link[];
  /** The credentials of every link request, as they crossed the wire. */
  linked: unknown[];
}

/**
 * Starts a local server that answers the session, methods, and provider link requests,
 * because no test Core accepts a provider. Every other request fails.
 * @returns The server, which closes when the test finishes.
 */
export const serveOIDC = async ({
  methods = ["password", "oidc"],
  providers = [],
  links = [],
  linkRefusal,
}: ServeOIDCParams = {}): Promise<OIDCServer> => {
  const state = { links: [...links], linked: [] as unknown[] };
  const answer = (path: string, body: string): { status?: number; body: unknown } => {
    if (path.endsWith("/auth/methods")) return { body: { methods, providers } };
    if (path.endsWith("/auth/session/create"))
      return { body: { token: "sys_local", session: SESSION } };
    if (path.endsWith("/auth/session/retrieve"))
      return { body: { sessions: [SESSION] } };
    if (path.endsWith("/auth/session/delete")) return { body: {} };
    if (path.endsWith("/auth/oidc/retrieve")) return { body: { links: state.links } };
    if (path.endsWith("/auth/oidc/unlink")) {
      const { keys } = JSON.parse(body) as { keys: string[] };
      state.links = state.links.filter((l) => !keys.includes(l.key));
      return { body: {} };
    }
    if (!path.endsWith("/auth/oidc/link")) return { status: 404, body: {} };
    const { credentials } = JSON.parse(body) as {
      credentials: auth.OIDCCredentials;
    };
    state.linked.push(credentials);
    if (linkRefusal != null)
      return { status: 400, body: { type: linkRefusal, data: "link refused" } };
    const issuer = providers.find((p) => p.provider === credentials.provider)?.issuer;
    const link = {
      key: crypto.randomUUID(),
      user: LOCAL_SUBJECT.key,
      issuer: issuer ?? credentials.provider,
      account: "new-account",
      label: "ada@example.com",
    };
    state.links.push(link);
    return { body: { link } };
  };
  const address = await listen((req, res) => {
    const chunks: Buffer[] = [];
    req.on("data", (chunk: Buffer) => chunks.push(chunk));
    req.on("end", () => {
      const path = req.url ?? "";
      const { status = 200, body } = answer(path, Buffer.concat(chunks).toString());
      res.statusCode = status;
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify(body));
    });
  });
  return {
    address,
    get links() {
      return state.links;
    },
    linked: state.linked,
  };
};

/**
 * Makes a client of the test Core ask a local server about providers and links, for a
 * spec that needs real permissions and a provider link together.
 */
export const pointAtOIDC = (client: Synnax, { address }: OIDCServer): void => {
  Object.assign(client.params, address);
  Object.assign(client.auth, {
    oidc: new auth.oidc.Client(
      auth.createTokenlessClient({ ...address, secure: false }),
    ),
  });
};
