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

import { readNodeTimeout } from "@/feature/opcua/device/commandTimeout";
import { type ConnectionConfig, ZERO_PROPERTIES } from "@/feature/opcua/device/types";

const config = (
  clientTimeoutMs: number,
  timeout = ZERO_PROPERTIES.connection.readNodeTimeout,
): ConnectionConfig => ({
  ...ZERO_PROPERTIES.connection,
  clientTimeoutMs,
  readNodeTimeout: timeout,
});

describe("readNodeTimeout", () => {
  it("should outlast the default node read timeout and two client timeouts", () => {
    expect(readNodeTimeout(config(5000)).valueOf()).toEqual(
      TimeSpan.seconds(32).valueOf(),
    );
  });

  it("should grow with the client timeout", () => {
    expect(readNodeTimeout(config(10_000)).valueOf()).toEqual(
      TimeSpan.seconds(42).valueOf(),
    );
  });

  it("should grow with the node read timeout of the connection", () => {
    const timeout = Number(TimeSpan.seconds(60).valueOf());
    expect(readNodeTimeout(config(5000, timeout)).valueOf()).toEqual(
      TimeSpan.seconds(72).valueOf(),
    );
  });
});
