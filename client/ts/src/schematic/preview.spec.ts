// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";

import { NotFoundError } from "@/index";
import { schematic } from "@/schematic";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const createPreview = (): schematic.Schematic =>
  schematic.schematicZ.parse({
    key: uuid.create(),
    name: "Engine at 3f9a1c7",
    snapshot: true,
    nodes: [{ key: "valve", position: { x: 10, y: 20 } }],
    configs: { valve: { variant: "valve", label: { label: "Fuel valve" } } },
  });

const expectNotFound = async (
  res: Promise<schematic.Schematic>,
  { key }: schematic.Schematic,
): Promise<void> => {
  await expect(res).rejects.toThrow(NotFoundError);
  await expect(res).rejects.toThrow(`schematic with key ${key} not found`);
};

describe("Schematic", () => {
  describe("preview", () => {
    it("should serve the previewed schematic from the cache", async () => {
      const preview = createPreview();
      onTestFinished(client.schematics.preview(preview));
      expect(await client.schematics.retrieve({ key: preview.key })).toEqual(preview);
    });

    it("should not create the schematic on the Core", async () => {
      const preview = createPreview();
      onTestFinished(client.schematics.preview(preview));
      const other = createTestClient();
      await expectNotFound(other.schematics.retrieve({ key: preview.key }), preview);
    });

    it("should drop the schematic from the cache when released", async () => {
      const preview = createPreview();
      const release = client.schematics.preview(preview);
      release();
      await expectNotFound(client.schematics.retrieve({ key: preview.key }), preview);
    });
  });
});
