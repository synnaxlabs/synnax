// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AstroIntegration } from "astro";

import config from "./astro.config.ts";

// The Vercel adapter cannot serve `astro preview`, so check-site builds and crawls the
// site with no adapter. With no adapter, every route must prerender.
const prerenderAll: AstroIntegration = {
  name: "prerender-all",
  hooks: {
    "astro:route:setup": ({ route }) => {
      route.prerender = true;
    },
  },
};

const checkConfig: typeof config = {
  ...config,
  adapter: undefined,
  integrations: [...(config.integrations ?? []), prerenderAll],
};

export default checkConfig;
