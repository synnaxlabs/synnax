// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Channel,
  manifestURL,
  type Product,
  type Releases,
} from "@/util/releases";

// Vercel's CDN serves the redirect for the window, so polling Consoles reach the
// function and the GitHub API at most once per window per region.
const CACHE_CONTROL = "public, s-maxage=300, stale-while-revalidate=60";

type Respond = (version: string) => Response;

const lookup = async (
  releases: Releases,
  product: Product,
  channel: Channel,
  respond: Respond,
): Promise<Response> => {
  try {
    return respond(await releases.latest(product, channel));
  } catch (e) {
    const message = e instanceof Error ? e.message : "release lookup failed";
    return new Response(message, {
      status: 503,
      headers: { "Cache-Control": "no-store" },
    });
  }
};

/**
 * Redirects to the manifest of the highest release of an app on the channel, or answers
 * 503 with the lookup failure so the updater retries on its next poll.
 * @param product - The app being polled.
 */
export const manifest = async (
  releases: Releases,
  product: Product,
  channel: Channel,
): Promise<Response> =>
  await lookup(
    releases,
    product,
    channel,
    (version) =>
      new Response(null, {
        status: 302,
        headers: {
          Location: manifestURL(product, version),
          "Cache-Control": CACHE_CONTROL,
        },
      }),
  );

/**
 * Answers the highest release version of a product on the channel as JSON, such as
 * `{"version":"0.61.0"}`, or 503 with the lookup failure. Any origin may read it, so a
 * Console in a browser can find a newer Core.
 */
export const version = async (
  releases: Releases,
  product: Product,
  channel: Channel,
): Promise<Response> =>
  await lookup(releases, product, channel, (found) =>
    Response.json(
      { version: found },
      {
        headers: {
          "Cache-Control": CACHE_CONTROL,
          "Access-Control-Allow-Origin": "*",
        },
      },
    ),
  );
