// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { Desktop } from "@/components/desktop";
import { assetURL } from "@/util/releases";

// The installer depends on the browser, so the route answers each request.
export const prerender = false;

/**
 * Redirects to the installer of the latest stable Synnax Desktop for the platform of
 * the browser, or to the install page when no installer fits it.
 */
export const GET: APIRoute = async ({ locals, request }) => {
  const version = await locals.releases.latest("desktop");
  const found = Desktop.installer(version, request.headers.get("user-agent") ?? "");
  const location =
    found == null
      ? "/reference/get-started/install-desktop"
      : assetURL("desktop", version, found.asset);
  return new Response(null, { status: 302, headers: { Location: location } });
};
