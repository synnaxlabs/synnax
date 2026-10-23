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

const ROUTES = Object.keys(import.meta.glob("/src/pages/**/*.astro")).map(
  (file) => file.replace(/^\/src\/pages/, "").replace(/(index)?\.astro$/, "") || "/",
);

export const GET = sitemap(ROUTES);
