// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createServer } from "node:http";
import { type AddressInfo } from "node:net";

import { type auth, type Synnax } from "@synnaxlabs/client";
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

const SESSION = {
  key: "8b7c3e52-6f0a-4c1d-9e2b-3a4f5d6e7f80",
  subject: { type: "user", key: "1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f" },
  credential: "2d3e4f5a-6b7c-4d8e-9f0a-1b2c3d4e5f60",
  lastActive: 0,
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
): Promise<auth.TokenlessParams & { port: number }> => {
  let methodsRequests = 0;
  const server = createServer((req, res) => {
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
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  onTestFinished(
    async () => await new Promise<void>((resolve) => server.close(() => resolve())),
  );
  return { host: "127.0.0.1", port: (server.address() as AddressInfo).port };
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
