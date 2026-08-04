// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connect, createServer, type Socket } from "node:net";

const DEFAULT_TARGET = { host: "localhost", port: 9090 };

export interface SeverableProxyTarget {
  host?: string;
  port?: number;
}

export interface SeverableProxy {
  /** Port the proxy accepts connections on. Point the client here. */
  port: number;
  /** Drops every live connection and refuses new ones, as if the cluster died. */
  sever: () => Promise<void>;
  /** Accepts connections again on the same port. */
  restore: () => Promise<void>;
  /** Shuts the proxy down for good. */
  close: () => Promise<void>;
}

/**
 * Creates a TCP proxy in front of the live test cluster so specs can simulate
 * cluster downtime without touching the cluster itself. Connect a client to
 * the returned port, then sever() and restore() the link.
 */
export const createSeverableProxy = async (
  target: SeverableProxyTarget = {},
): Promise<SeverableProxy> => {
  const { host, port } = { ...DEFAULT_TARGET, ...target };
  const sockets = new Set<Socket>();
  const server = createServer((downstream) => {
    const upstream = connect(port, host);
    sockets.add(downstream).add(upstream);
    const destroy = (): void => {
      sockets.delete(downstream);
      sockets.delete(upstream);
      downstream.destroy();
      upstream.destroy();
    };
    downstream.on("error", destroy);
    upstream.on("error", destroy);
    downstream.on("close", destroy);
    upstream.on("close", destroy);
    downstream.pipe(upstream);
    upstream.pipe(downstream);
  });
  const listen = async (listenPort: number): Promise<void> =>
    await new Promise((resolve, reject) => {
      server.once("error", reject);
      server.listen(listenPort, () => {
        server.removeListener("error", reject);
        resolve();
      });
    });
  await listen(0);
  const address = server.address();
  if (address == null || typeof address === "string")
    throw new Error("proxy failed to bind a port");
  const boundPort = address.port;
  const sever = async (): Promise<void> => {
    const closed = new Promise<void>((resolve) => server.close(() => resolve()));
    sockets.forEach((socket) => socket.destroy());
    sockets.clear();
    await closed;
  };
  return {
    port: boundPort,
    sever,
    restore: async () => await listen(boundPort),
    close: sever,
  };
};
