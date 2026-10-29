// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIContext, type APIRoute } from "astro";

import { modeFor } from "../consent";

export const prerender = false;

/** Responds with the cookie mode of the visitor's country: `ask` or `notice`. */
export const GET = (({ request }: Pick<APIContext, "request">) =>
  new Response(modeFor(request.headers.get("x-vercel-ip-country")), {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      // The answer differs per visitor, so only their browser may keep it. A day saves
      // a request on each page for a visitor who ignores the card.
      "Cache-Control": "private, max-age=86400",
    },
  })) satisfies APIRoute;
