// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AstroUserConfig } from "astro";
import { envField } from "astro/config";

/** The variables the site reads from `astro:env`. The site and its specs share them. */
export const schema: NonNullable<AstroUserConfig["env"]>["schema"] = {
  // Vercel sets this at build time; flags read it to light up preview deploys.
  VERCEL_ENV: envField.string({ context: "client", access: "public", optional: true }),
};
