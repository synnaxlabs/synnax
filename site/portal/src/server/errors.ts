// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * HTTPError carries the status a route answers with when it rejects a request. A `code`
 * names a refusal that a client acts on, so it need not match the message.
 */
export class HTTPError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly code?: string,
    options?: ErrorOptions,
  ) {
    super(message, options);
    this.name = "HTTPError";
  }
}

export const notFound = (what: string): HTTPError =>
  new HTTPError(404, `${what} not found`);
export const forbidden = (message = "Forbidden", code?: string): HTTPError =>
  new HTTPError(403, message, code);
export const unauthorized = (): HTTPError => new HTTPError(401, "Log in first");
export const badRequest = (message: string, options?: ErrorOptions): HTTPError =>
  new HTTPError(400, message, undefined, options);
export const tooMany = (): HTTPError =>
  new HTTPError(429, "Too many activations. Try again later.");

/**
 * toResponse turns a thrown error into the response a JSON route returns: `{ error }`,
 * or `{ error, code }` when the error has a code.
 */
export const toResponse = (err: unknown): Response => {
  if (err instanceof HTTPError)
    return Response.json(
      { error: err.message, code: err.code },
      { status: err.status },
    );
  console.error(err);
  return Response.json({ error: "Internal error" }, { status: 500 });
};
