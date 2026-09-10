// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import config from "./astro.config.ts";

// The Vercel adapter cannot serve `astro preview`, so check-site builds and crawls
// the site statically. Everything else matches the deployed config.
const checkConfig: typeof config = { ...config, output: "static", adapter: undefined };

export default checkConfig;
