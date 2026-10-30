// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { commandTimeout } from "@/feature/opcua/device/commandTimeout";
import { ZERO_PROPERTIES } from "@/feature/opcua/device/types";

describe("commandTimeout", () => {
  it("should wait twice the client timeout and a margin", () => {
    const connection = { ...ZERO_PROPERTIES.connection, clientTimeoutMs: 20_000 };
    expect(commandTimeout(connection).equals(TimeSpan.seconds(42))).toBe(true);
  });
});
