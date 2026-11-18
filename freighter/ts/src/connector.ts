// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { runtime } from "@synnaxlabs/x";

/** The request fields a transport sends with. */
export interface FetchInit {
  method: string;
  body: Uint8Array<ArrayBuffer> | ReadableStream<Uint8Array> | Blob;
  headers: Record<string, string>;
  signal?: AbortSignal;
  duplex?: "half";
}

/** The response fields a transport reads. */
export interface FetchResponse {
  readonly ok: boolean;
  readonly status: number;
  readonly statusText: string;
  arrayBuffer: () => Promise<ArrayBuffer>;
  readonly body: ReadableStream<Uint8Array> | null;
}

/** Sends an HTTP request, like the global fetch. */
export type Fetch = (target: string, init: FetchInit) => Promise<FetchResponse>;

// TypeScript compares the event both ways, so the DOM and undici WebSockets both fit.
type Listener<E> = { bivarianceHack(ev: E): void }["bivarianceHack"];

/** The WebSocket members a stream uses. */
export interface Socket {
  binaryType: string;
  readonly bufferedAmount: number;
  readonly readyState: number;
  send: (data: Uint8Array<ArrayBuffer>) => void;
  close: () => void;
  onopen: Listener<unknown> | null;
  onmessage: Listener<{ data: ArrayBuffer }> | null;
  onclose: Listener<{ code: number }> | null;
  onerror: Listener<object> | null;
}

/** Opens a WebSocket to the URL and sends the headers with the upgrade request. */
export type Dial = (url: string, headers: Record<string, string>) => Promise<Socket>;

/** The functions a transport opens its connections with. */
export interface Connector {
  fetch: Fetch;
  dial: Dial;
  /**
   * True when dial sends the Authorization header to the server. Otherwise the client
   * puts it in the URL.
   */
  sendsHeaders?: boolean;
}

// Node, Bun, and Deno take a headers option that the DOM types do not have.
const openWithOptions = (url: string, options: object): Socket => {
  const ws: unknown = Reflect.construct(WebSocket, [url, options]);
  if (!(ws instanceof WebSocket)) throw new Error("WebSocket did not construct");
  return ws;
};

const SUBPROTOCOL = "synnax.v1";
const TOKEN_SUBPROTOCOL_PREFIX = "synnax.token.";
const BEARER_PREFIX = "Bearer ";

// A session token or API key holds only base64url characters, which a subprotocol
// allows.
const openWithSubprotocol = (url: string, authorization?: string): Socket => {
  if (authorization == null) return new WebSocket(url);
  if (!authorization.startsWith(BEARER_PREFIX))
    throw new Error("WebSocket authorization must be a Bearer token");
  const token = authorization.slice(BEARER_PREFIX.length);
  return new WebSocket(url, [SUBPROTOCOL, `${TOKEN_SUBPROTOCOL_PREFIX}${token}`]);
};

/**
 * Opens connections with the runtime's global fetch and WebSocket. A DOM WebSocket
 * cannot send headers, so it sends only the Authorization header, as a subprotocol.
 */
export const DEFAULT_CONNECTOR: Connector = {
  fetch: async (target, init) => await fetch(target, init),
  // An Electron renderer or jsdom page sets process.versions.node but has a DOM
  // WebSocket.
  dial: async (url, headers) =>
    runtime.RUNTIME === "node" && typeof document === "undefined"
      ? openWithOptions(url, { headers })
      : openWithSubprotocol(url, headers.Authorization),
  sendsHeaders: true,
};

// A CA list replaces the default list in Node and Bun, so it must keep the defaults.
const withDefaultCAs = async (ca: string): Promise<string[]> => {
  const { getCACertificates, rootCertificates } = await import("node:tls");
  // Node 22.15 added getCACertificates, which also keeps NODE_EXTRA_CA_CERTS.
  const defaults =
    getCACertificates == null ? rootCertificates : getCACertificates("default");
  return [...defaults, ca];
};

const loadUndici = async (ca: string) => {
  const { Agent, fetch, WebSocket } = await import("undici");
  const connect = { ca: await withDefaultCAs(ca) };
  return { fetch, WebSocket, dispatcher: new Agent({ connect }) };
};

const nodeConnector = (ca: string): Connector => {
  // Node's built-in fetch and WebSocket reject an Agent from the undici package, so
  // the connections use the package's own fetch and WebSocket. The import waits for
  // the first connection, so a client that never connects never loads undici.
  let undici: ReturnType<typeof loadUndici> | null = null;
  const load = async () => await (undici ??= loadUndici(ca));
  return {
    fetch: async (target, { body, ...init }) => {
      const { fetch, dispatcher } = await load();
      const res = await fetch(target, {
        ...init,
        body: body instanceof Blob ? body.stream() : body,
        dispatcher,
      });
      return {
        ok: res.ok,
        status: res.status,
        statusText: res.statusText,
        arrayBuffer: async () => await res.arrayBuffer(),
        // Undici's body is the global ReadableStream. The check narrows its type.
        body: res.body instanceof ReadableStream ? res.body : null,
      };
    },
    dial: async (url, headers) => {
      const { WebSocket, dispatcher } = await load();
      return new WebSocket(url, { dispatcher, headers });
    },
    sendsHeaders: true,
  };
};

const bunConnector = (ca: string): Connector => {
  let cas: Promise<string[]> | null = null;
  const load = async () => ({ ca: await (cas ??= withDefaultCAs(ca)) });
  return {
    fetch: async (target, init) => {
      const req: RequestInit & { tls: object } = { ...init, tls: await load() };
      return await fetch(target, req);
    },
    dial: async (url, headers) => openWithOptions(url, { headers, tls: await load() }),
    sendsHeaders: true,
  };
};

const denoConnector = (deno: object, ca: string): Connector => {
  const createHttpClient: unknown = Reflect.get(deno, "createHttpClient");
  if (typeof createHttpClient !== "function")
    throw new Error("Deno.createHttpClient is not available");
  const client: unknown = Reflect.apply(createHttpClient, deno, [{ caCerts: [ca] }]);
  return {
    fetch: async (target, init) => {
      const req: RequestInit & { client: unknown } = { ...init, client };
      return await fetch(target, req);
    },
    dial: async () => {
      throw new Error(
        "Deno cannot trust a CA certificate for one WebSocket. Run Deno with " +
          "--cert <file>, or set DENO_CERT to the CA certificate file.",
      );
    },
  };
};

/**
 * Returns a connector that trusts the CA certificate and the runtime's default
 * certificates. In Deno, the connector's dial throws, because a Deno WebSocket cannot
 * trust a CA certificate of its own.
 * @param caCert - The PEM text of the CA certificate.
 * @throws {Error} in a browser, which trusts only the operating system or browser
 * certificate store.
 */
export const trustCA = (caCert: string): Connector => {
  const deno: unknown = Reflect.get(globalThis, "Deno");
  if (typeof deno === "object" && deno !== null) return denoConnector(deno, caCert);
  if ("Bun" in globalThis) return bunConnector(caCert);
  if (typeof process !== "undefined" && process.versions?.node != null)
    return nodeConnector(caCert);
  throw new Error(
    "A browser cannot trust a CA certificate for one client. Add the CA " +
      "certificate to the operating system or browser certificate store.",
  );
};
