// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";

import { type Core, SLICE_NAME } from "@/session/core/slice";
import { type ConsolePreloadedState } from "@/testutil";

/** Connection parameters for the local test core every live-core spec runs against. */
export const CONNECTION_PARAMS: Omit<Core, "key" | "name"> = {
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
};

export const createCore = (name: string, overrides: Partial<Core> = {}): Core => ({
  key: uuid.create(),
  name,
  ...CONNECTION_PARAMS,
  ...overrides,
});

export const createCoreState = (
  cores: Core[],
  selected?: string,
): ConsolePreloadedState => ({
  [SLICE_NAME]: {
    version: 0,
    selected,
    cores: Object.fromEntries(cores.map((c) => [c.key, c])),
  },
});
