// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sitemap } from "@synnaxlabs/site-common/sitemap";

export const prerender = true;

// The 404 page answers every path that has no page, so it has no address of its own.
const ROUTES = Object.keys(import.meta.glob("/src/pages/**/*.astro"))
  .map(
    (file) => file.replace(/^\/src\/pages/, "").replace(/(index)?\.astro$/, "") || "/",
  )
  .filter((route) => route !== "/404");

export const GET = sitemap(ROUTES);
