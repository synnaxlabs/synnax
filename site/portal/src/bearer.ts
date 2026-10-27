// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { unauthorized } from "@/server/errors";

const BEARER = /^Bearer\s+(\S+)$/i;

// The Desktop app calls from its own origin, so its routes answer preflights and
// mark every response, refusals included, for any origin. The secret is the guard.
const CORS_HEADERS = {
  "access-control-allow-origin": "*",
  "access-control-allow-methods": "POST, OPTIONS",
  "access-control-allow-headers": "authorization",
};

/** withCORS marks a response for any origin. */
export const withCORS = (res: Response): Response => {
  for (const [name, value] of Object.entries(CORS_HEADERS))
    res.headers.set(name, value);
  return res;
};

/** preflight answers the CORS preflight of a route the Desktop app calls. */
export const preflight: APIRoute = () =>
  new Response(null, { status: 204, headers: CORS_HEADERS });

/**
 * readSecret reads the renewal secret that a request carries as its bearer token. Throws
 * a 401 when the request carries none.
 */
export const readSecret = (request: Request): string => {
  const token = BEARER.exec(request.headers.get("authorization") ?? "")?.[1];
  if (token == null) throw unauthorized();
  return token;
};
