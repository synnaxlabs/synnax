// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// src/types/astro-components.d.ts
declare module "*.astro" {
  import type { AstroComponentFactory } from "astro/runtime";
  const component: AstroComponentFactory;
  export default component;
}
