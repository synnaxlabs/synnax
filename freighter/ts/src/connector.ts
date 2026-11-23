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

/**
 * Opens a WebSocket to the URL and sends the headers with the upgrade request. A dial
 * that cannot send headers sends the Authorization header as the {@link subprotocols}.
 */
export type Dial = (url: string, headers: Record<string, string>) => Promise<Socket>;

/** The functions a transport opens its connections with. */
export interface Connector {
  fetch: Fetch;
  dial: Dial;
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

/**
 * Returns the WebSocket subprotocols that carry the Authorization header to the Core.
 * @param authorization - The Authorization header value, or undefined for none.
 * @throws {Error} if the authorization is not a Bearer token.
 */
export const subprotocols = (authorization?: string): string[] => {
  if (authorization == null) return [];
  if (!authorization.startsWith(BEARER_PREFIX))
    throw new Error("WebSocket authorization must be a Bearer token");
  // A session token or API key holds only base64url characters, which a subprotocol
  // allows.
  const token = authorization.slice(BEARER_PREFIX.length);
  return [SUBPROTOCOL, `${TOKEN_SUBPROTOCOL_PREFIX}${token}`];
};

// A DOM WebSocket cannot send headers, so it sends only the Authorization header, as a
// subprotocol.
const GLOBAL_CONNECTOR: Connector = {
  fetch: async (target, init) => await fetch(target, init),
  // An Electron renderer or jsdom page sets process.versions.node but has a DOM
  // WebSocket.
  dial: async (url, headers) =>
    runtime.RUNTIME === "node" && typeof document === "undefined"
      ? openWithOptions(url, { headers })
      : new WebSocket(url, subprotocols(headers.Authorization)),
};

// A CA list replaces the default list in Node and Bun, so it must keep the defaults.
const withDefaultCAs = async (ca: string): Promise<string[]> => {
  // webpack cannot build a node: import for a browser, but it skips getBuiltinModule.
  const { getCACertificates, rootCertificates } = process.getBuiltinModule("node:tls");
  // Node 22.15 added getCACertificates. Before it, the defaults are the bundled roots
  // and NODE_EXTRA_CA_CERTS.
  if (getCACertificates != null) return [...getCACertificates("default"), ca];
  const extra = process.env.NODE_EXTRA_CA_CERTS;
  if (extra == null || extra === "") return [...rootCertificates, ca];
  const { readFile } = process.getBuiltinModule("node:fs/promises");
  // Node warns and skips an unreadable NODE_EXTRA_CA_CERTS file, so this skips it too.
  const extraCAs = await readFile(extra, "utf8").catch(() => null);
  return extraCAs == null
    ? [...rootCertificates, ca]
    : [...rootCertificates, extraCAs, ca];
};

const loadUndici = async (ca?: string) => {
  const { EnvHttpProxyAgent, fetch, WebSocket } = await import("undici");
  // The agent uses connect for a direct connection and requestTls through a proxy.
  const tls = ca == null ? undefined : { ca: await withDefaultCAs(ca) };
  const dispatcher = new EnvHttpProxyAgent({ connect: tls, requestTls: tls });
  return { fetch, WebSocket, dispatcher };
};

const PROXY_VARIABLES: Array<[string, string]> = [
  ["http_proxy", "HTTP_PROXY"],
  ["https_proxy", "HTTPS_PROXY"],
];

// Undici reads the lowercase name first, and an empty value sets no proxy.
const proxyVariables = (): Array<[string, string]> =>
  PROXY_VARIABLES.flatMap(([lower, upper]): Array<[string, string]> => {
    const name = process.env[lower] == null ? upper : lower;
    const value = process.env[name];
    return value == null || value === "" ? [] : [[name, value]];
  });

// URL reads the host of "host:port" as a scheme, so the protocol must be checked.
const isProxyURL = (value: string): boolean =>
  URL.canParse(value) && ["http:", "https:"].includes(new URL(value).protocol);

const nodeConnector = (ca?: string): Connector => {
  // Undici throws for an invalid proxy URL only at the first connection, and without
  // the name of the variable.
  for (const [name, value] of proxyVariables())
    if (!isProxyURL(value))
      throw new Error(`${name} is not a valid proxy URL: "${value}"`);
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

// Bun and Deno read the proxy variables themselves. Node's global fetch does not.
const usesNodeProxy = (): boolean =>
  runtime.RUNTIME === "node" &&
  typeof document === "undefined" &&
  !("Bun" in globalThis) &&
  !("Deno" in globalThis) &&
  proxyVariables().length > 0;

/**
 * Returns the connector a client uses when it is given none: the runtime's global
 * fetch and WebSocket. In Node, when HTTP_PROXY or HTTPS_PROXY is set, the connector
 * sends connections through that proxy, except to the hosts in NO_PROXY.
 * @throws {Error} in Node, if a proxy variable is not a valid URL.
 */
export const defaultConnector = (): Connector =>
  usesNodeProxy() ? nodeConnector() : GLOBAL_CONNECTOR;

/**
 * Returns a connector that trusts the CA certificate and the runtime's default
 * certificates. In Node, the connector uses the proxy that HTTP_PROXY, HTTPS_PROXY, and
 * NO_PROXY name. In Deno, the connector's dial throws, because a Deno WebSocket cannot
 * trust a CA certificate of its own.
 * @param caCert - The PEM text of the CA certificate.
 * @throws {Error} in a browser, which trusts only the operating system or browser
 * certificate store.
 * @throws {Error} in Node, if a proxy variable is not a valid URL.
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
