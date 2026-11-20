// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import react from "@astrojs/react";
import vercel from "@astrojs/vercel";
import { integration } from "@synnaxlabs/site-common/integration";
import { defineConfig } from "astro/config";

export default defineConfig({
  integrations: [integration(), react()],
  adapter: vercel(),
  vite: {
    // Prebundle the island's linked workspace modules so their icon dependencies
    // are ready before the browser hydrates it.
    optimizeDeps: {
      include: [
        "@synnaxlabs/lyra/button",
        "@synnaxlabs/lyra/icon",
        "@synnaxlabs/lyra/logo",
      ],
    },
  },
  site: "https://foundation.synnaxlabs.com",
});
