// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createServer, type RequestListener } from "node:http";
import { type AddressInfo } from "node:net";

import { auth, type ontology, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type RenderResult } from "@testing-library/react";
import { type ReactNode } from "react";
import { onTestFinished } from "vitest";

import { Modals } from "@/platform/modals";
import {
  type ConsolePreloadedState,
  renderWithConsole,
  type TestStore,
} from "@/testutil";

/**
 * Renders ui together with a live {@link Modals.Stack} inside the full console provider
 * stack, so modals opened during the test actually mount. Returns the backing store so
 * a spec can inspect dispatched Core state.
 */
export const renderCoreUI = async (
  ui: ReactNode,
  preloadedState?: ConsolePreloadedState,
): Promise<RenderResult & { store: TestStore }> =>
  await renderWithConsole(
    <>
      {ui}
      <Modals.Stack />
    </>,
    { preloadedState },
  );

/** The subject of every session a local auth server opens. */
export const LOCAL_SUBJECT: ontology.ID = {
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

/**
 * Starts a local server that answers the methods request, because no test Core turns
 * passwords off or accepts a provider. The Console asks once for the methods and again
 * for the providers; null providers fail that second request. A token opens a session
 * for any code, and without one the server rejects every code.
 * @returns The address of the server, which closes when the test finishes.
 */
export const serveAuth = async (
  methods: string[],
  providers: auth.OIDCLogin[] | null = [],
  onServed?: () => void,
  token?: string,
): Promise<LocalAddress> => {
  let methodsRequests = 0;
  return await listen((req, res) => {
    req.resume();
    req.on("end", () => {
      if (token != null && req.url?.endsWith("/auth/session/create") === true) {
        res.setHeader("Content-Type", "application/json");
        res.end(JSON.stringify({ token, session: SESSION }));
        return;
      }
      const isMethods = req.url?.endsWith("/auth/methods") === true;
      if (isMethods) methodsRequests += 1;
      if (!isMethods || (providers == null && methodsRequests > 1)) {
        res.statusCode = 404;
        res.end();
        return;
      }
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify({ methods, providers: providers ?? [] }), onServed);
    });
  });
};

export interface ServeOIDCParams {
  methods?: string[];
  providers?: auth.OIDCLogin[];
  links?: auth.oidc.Link[];
  /** Refuses every link, as for an account that already has one. */
  linkRefused?: boolean;
  /** Called after each methods answer. */
  onServed?: () => void;
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
  linkRefused = false,
  onServed,
}: ServeOIDCParams = {}): Promise<OIDCServer> => {
  const state = { links: [...links], linked: [] as unknown[] };
  const answer = (path: string, body: string): { status?: number; body: unknown } => {
    if (path.endsWith("/auth/methods")) return { body: { methods, providers } };
    if (path.endsWith("/auth/session/create"))
      return { body: { token: "sys_local", session: SESSION } };
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
    if (linkRefused)
      return {
        status: 400,
        body: { type: "sy.validation", data: "the account is already linked" },
      };
    const link = {
      key: crypto.randomUUID(),
      subject: LOCAL_SUBJECT,
      provider: credentials.provider,
      account: "new-account",
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
      res.end(JSON.stringify(body), () => {
        if (path.endsWith("/auth/methods")) onServed?.();
      });
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
    oidc: new auth.oidc.Client(auth.createTokenlessClient(address)),
  });
};

/**
 * Creates a client of the test Core that reports a local server as its address, so a
 * spec sees a Core that accepts only the given methods. Everything else reaches the
 * test Core.
 */
export const createClientAccepting = async (methods: string[]): Promise<Synnax> => {
  const client = createTestClient();
  Object.assign(client.params, await serveAuth(methods));
  return client;
};
