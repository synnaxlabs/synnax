// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

/** Opens a WebSocket to the URL. */
export type Dial = (url: string) => Promise<Socket>;

/** The functions a transport opens its connections with. */
export interface Connector {
  fetch: Fetch;
  dial: Dial;
}

/** Opens connections with the runtime's global fetch and WebSocket. */
export const DEFAULT_CONNECTOR: Connector = {
  fetch: async (target, init) => await fetch(target, init),
  dial: async (url) => new WebSocket(url),
};

const loadUndici = async (ca: string) => {
  const { Agent, fetch, WebSocket } = await import("undici");
  return { fetch, WebSocket, dispatcher: new Agent({ connect: { ca } }) };
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
    dial: async (url) => {
      const { WebSocket, dispatcher } = await load();
      return new WebSocket(url, { dispatcher });
    },
  };
};

const bunConnector = (ca: string): Connector => ({
  fetch: async (target, init) => {
    const req: RequestInit & { tls: { ca: string } } = { ...init, tls: { ca } };
    return await fetch(target, req);
  },
  dial: async (url) => {
    // The DOM types do not have Bun's options argument.
    const ws: unknown = Reflect.construct(WebSocket, [url, { tls: { ca } }]);
    if (!(ws instanceof WebSocket)) throw new Error("Bun did not open a WebSocket");
    return ws;
  },
});

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
