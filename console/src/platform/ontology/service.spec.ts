// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Ontology } from "@/platform/ontology";

describe("NOOP_SERVICE", () => {
  it("should report that it has children by default", () => {
    expect(Ontology.NOOP_SERVICE.hasChildren).toBe(true);
  });

  it("should never accept a drop", () => {
    const source = { type: "channel", key: "1" };
    expect(Ontology.NOOP_SERVICE.canDrop({ source, items: [] })).toBe(false);
  });

  it("should produce no haul items for any resource", () => {
    const resource: ontology.Resource = {
      key: "channel:1",
      id: ontology.idZ.parse("channel:1"),
      name: "test",
    };
    expect(Ontology.NOOP_SERVICE.haulItems(resource)).toEqual([]);
  });

  it("should not define an onSelect handler", () => {
    expect(Ontology.NOOP_SERVICE.onSelect).toBeUndefined();
  });
});
