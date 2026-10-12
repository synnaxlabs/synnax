// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { blob, ordered } from "@/util/reliability";

export const prerender = true;

/**
 * Test details for the reliability wall's hover card, in wall order. Sources are paths
 * relative to `blob`, which is sent once. Tests index into the few distinct tag lists.
 */
export const GET: APIRoute = () => {
  const suites = [...new Set(ordered.map((t) => t.suite))];
  const index = new Map(suites.map((s, i) => [s, i]));
  const tags = [...new Set(ordered.map((t) => t.tags.join(" ")))];
  const tagIndex = new Map(tags.map((t, i) => [t, i]));
  return new Response(
    JSON.stringify({
      blob,
      suites: suites.map((s) => ({ job: s.job, os: s.os })),
      tags: tags.map((t) => t.split(" ")),
      tests: ordered.map((t) => [
        index.get(t.suite),
        t.name,
        t.duration,
        t.source.slice(blob.length),
        tagIndex.get(t.tags.join(" ")),
      ]),
    }),
    { headers: { "Content-Type": "application/json" } },
  );
};
