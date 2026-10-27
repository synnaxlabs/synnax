// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { mergeConfig } from "vite";

import config from "./vite.config.ts";

// Built workspace packages load as native modules. The module runner wraps every export
// in a getter, which adds a cost to each call that a production bundle does not pay.
// Pluto's own modules still pay about 40 ns per access, so their warning is off.
export default mergeConfig(config, {
  test: {
    server: { deps: { external: [/\/ts\/dist\//] } },
    benchmark: { suppressExportGetterWarnings: true },
  },
});
