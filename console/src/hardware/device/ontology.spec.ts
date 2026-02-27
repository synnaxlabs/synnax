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

const mockResource = (data?: Record<string, unknown>): ontology.Resource =>
  ({
    id: { type: "device", key: "1" },
    name: "test-device",
    data,
  }) as unknown as ontology.Resource;

describe("Device ONTOLOGY_SERVICE hasChildren", () => {
  it("should return true when isChassis is true", () => {
    const resource = mockResource({ properties: { isChassis: true } });
    expect(resolveHasChildren(ONTOLOGY_SERVICE, resource)).toBe(true);
  });

  it("should return false when isChassis is false", () => {
    const resource = mockResource({ properties: { isChassis: false } });
    expect(resolveHasChildren(ONTOLOGY_SERVICE, resource)).toBe(false);
  });

  it("should return false when isChassis is not present", () => {
    const resource = mockResource({ properties: {} });
    expect(resolveHasChildren(ONTOLOGY_SERVICE, resource)).toBe(false);
  });

  it("should return false when properties is null", () => {
    const resource = mockResource({ properties: null });
    expect(resolveHasChildren(ONTOLOGY_SERVICE, resource)).toBe(false);
  });

  it("should return false when data has no properties", () => {
    const resource = mockResource({});
    expect(resolveHasChildren(ONTOLOGY_SERVICE, resource)).toBe(false);
  });

  it("should return false when data is undefined", () => {
    const resource = mockResource(undefined);
    expect(resolveHasChildren(ONTOLOGY_SERVICE, resource)).toBe(false);
  });
});
