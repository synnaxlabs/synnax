// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connect, createServer, type Socket } from "node:net";

import { type breaker, TimeSpan } from "@synnaxlabs/x";

const DEFAULT_TARGET = { host: "localhost", port: 9090 };

/**
 * Retry policy for a client pointed at a severable proxy. A spec drives its own
 * retries, so the client's must finish well inside one assertion budget: the
 * default policy waits seconds per request and outlasts a single `expect.poll`
 * attempt, which reports as a timeout rather than the real answer.
 */
export const FAST_RETRY: breaker.Config = {
  baseInterval: TimeSpan.milliseconds(10),
  maxInterval: TimeSpan.milliseconds(50),
  scale: 1.5,
};

export interface SeverableProxyTarget {
  host?: string;
  port?: number;
}

export interface SeverableProxy {
  /** Port the proxy accepts connections on. Point the client here. */
  port: number;
  /** Drops every live connection and refuses new ones, as if the cluster died. */
  sever: () => Promise<void>;
  /**
   * Silently kills every live WebSocket connection, modeling a half-open socket (NAT
   * timeout, VPN drop, laptop sleep): the cluster side is destroyed while the client
   * side is held open with its writes discarded, so no close or reset ever reaches the
   * client. New connections still forward. Returns how many connections were
   * blackholed.
   */
  blackholeStreams: () => number;
  /** Accepts connections again on the same port. */
  restore: () => Promise<void>;
  /** Shuts the proxy down for good. */
  close: () => Promise<void>;
}

interface ProxiedPair {
  downstream: Socket;
  upstream: Socket;
  /** Whether a WebSocket upgrade was observed on this connection. */
  ws: boolean;
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
  const pairs = new Set<ProxiedPair>();
  // Client-side sockets of blackholed connections: held open, silent, until the proxy
  // is severed or closed.
  const zombies = new Set<Socket>();
  let severed = false;
  const server = createServer((downstream) => {
    // A connection the kernel accepted before the sever is still delivered after it,
    // and forwarding it would let a request through a severed link.
    if (severed) {
      downstream.destroy();
      return;
    }
    const upstream = connect(port, host);
    const pair: ProxiedPair = { downstream, upstream, ws: false };
    pairs.add(pair);
    // The upgrade can arrive on any request of a pooled keep-alive connection, not just
    // the first, so every chunk is sniffed until one is seen.
    downstream.on("data", (chunk: Buffer) => {
      if (!pair.ws && /upgrade:\s*websocket/i.test(chunk.toString("latin1")))
        pair.ws = true;
    });
    const destroy = (): void => {
      pairs.delete(pair);
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
    severed = true;
    const closed = new Promise<void>((resolve) => server.close(() => resolve()));
    pairs.forEach(({ downstream, upstream }) => {
      downstream.destroy();
      upstream.destroy();
    });
    pairs.clear();
    zombies.forEach((socket) => socket.destroy());
    zombies.clear();
    await closed;
  };
  const blackholeStreams = (): number => {
    let count = 0;
    for (const pair of [...pairs]) {
      if (!pair.ws) continue;
      count += 1;
      const { downstream, upstream } = pair;
      downstream.unpipe(upstream);
      upstream.unpipe(downstream);
      for (const socket of [downstream, upstream]) {
        socket.removeAllListeners("close");
        socket.removeAllListeners("error");
        socket.on("error", () => {});
      }
      // Discard anything the client still writes so backpressure cannot close the
      // socket on the proxy's behalf.
      downstream.on("data", () => {});
      downstream.resume();
      upstream.destroy();
      pairs.delete(pair);
      zombies.add(downstream);
    }
    return count;
  };
  return {
    port: boundPort,
    sever,
    blackholeStreams,
    restore: async () => {
      severed = false;
      await listen(boundPort);
    },
    close: sever,
  };
};
