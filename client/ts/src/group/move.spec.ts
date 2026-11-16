// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { device } from "@/device";
import { group } from "@/group";
import { project } from "@/project";
import { rack } from "@/rack";
import { type resource } from "@/resource";
import { schematic } from "@/schematic";

describe("canMove", () => {
  const destination = group.resourceID("g");

  it("should move a rack or project only into a group at the top", () => {
    const cases: [resource.ID, resource.ID, resource.ID][] = [
      [rack.resourceID(2), device.TYPE_RESOURCE_ID, rack.resourceID(1)],
      [project.resourceID("b"), project.TYPE_RESOURCE_ID, project.resourceID("a")],
    ];
    for (const [id, root, holder] of cases) {
      const moved = [{ id, ancestors: [root] }];
      expect(group.canMove({ root, destination, ancestors: [root], moved })).toBe(true);
      expect(
        group.canMove({ root, destination, ancestors: [holder, root], moved }),
      ).toBe(false);
    }
  });

  describe("device", () => {
    const root = device.TYPE_RESOURCE_ID;
    const canMove = (from: resource.ID[], to: resource.ID[]): boolean =>
      group.canMove({
        root,
        destination,
        ancestors: [...to, root],
        moved: [{ id: device.resourceID("d"), ancestors: [...from, root] }],
      });

    it("should keep a device under its rack", () => {
      expect(canMove([rack.resourceID(1)], [rack.resourceID(1)])).toBe(true);
      expect(canMove([rack.resourceID(1)], [rack.resourceID(2)])).toBe(false);
    });

    it("should keep a device held by another device in place", () => {
      const from = [device.resourceID("p"), rack.resourceID(1)];
      expect(canMove(from, [rack.resourceID(1)])).toBe(false);
    });
  });

  it("should move a record that is not pinned into a group under a project", () => {
    const root = project.TYPE_RESOURCE_ID;
    const ancestors = [project.resourceID("a"), root];
    const canMove = (id: resource.ID): boolean =>
      group.canMove({
        root,
        destination,
        ancestors,
        moved: [{ id, ancestors: [root] }],
      });
    expect(canMove(schematic.resourceID("s"))).toBe(true);
    expect(canMove(project.resourceID("b"))).toBe(false);
  });
});
