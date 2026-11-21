// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll } from "vitest";

/**
 * Loads the modules of lazy components before the specs of a file run, so a cold load
 * counts against the hook timeout and not the first spec that renders one. Call it
 * once, at the top of a spec file or from a domain's preload helper.
 */
export const preload = (...loads: Array<() => Promise<unknown>>): void =>
  beforeAll(async () => {
    await Promise.all(loads.map(async (load) => await load()));
  });
