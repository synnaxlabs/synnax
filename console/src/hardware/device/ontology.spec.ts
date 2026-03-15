// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { ONTOLOGY_SERVICE } from "@/hardware/device/ontology";
import { resolveHasChildren } from "@/ontology/service";

const mockResource = (data?: Record<string, unknown>): ontology.Resource => ({
  key: "device:1",
  id: { type: "device", key: "1" },
  name: "test-device",
  data,
});

describe("Device ONTOLOGY_SERVICE hasChildren", () => {
  it("should always return true for any device", () => {
    expect(resolveHasChildren(ONTOLOGY_SERVICE, mockResource())).toBe(true);
  });
});
