// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AstroUserConfig } from "astro";

import { docs } from "./astro.config.ts";

// The Vercel adapter cannot serve `astro preview`, so check-site builds and crawls
// the site statically. The portal is left out: its pages need a session and a
// database, and none of them is crawled.
const checkConfig: AstroUserConfig = { ...docs, output: "static", adapter: undefined };

export default checkConfig;
