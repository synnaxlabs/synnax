// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import { panel } from "@/panel";
import { type project } from "@/project";
import { type query } from "@/query";
import { createPanelProject, createTestClient, expectLive } from "@/testutil";

const client = createTestClient();

describe("Panel Client", () => {
  // Every panel here shares this project.
  let proj: project.Key;

  beforeAll(async () => {
    proj = await createPanelProject(client);
  });

  const createPanel = async (): Promise<panel.Panel> =>
    await client.panels.create({ name: `panel-${id.create()}`, project: proj });

  describe("retrieve", () => {
    it("should combine a project filter with a where", async () => {
      const scope = await createPanelProject(client);
      const [kept] = await client.panels.create([
        { name: `panel-${id.create()}`, project: scope },
        { name: `panel-${id.create()}`, project: scope },
      ]);
      const outsider = await createPanel();
      const retrieved = await client.panels.retrieve({
        where: { projects: [scope], key: { in: [kept.key, outsider.key] } },
      });
      expect(retrieved.map((p) => p.key)).toEqual([kept.key]);
    });

    it("should combine a project filter with a flat filter", async () => {
      const scope = await createPanelProject(client);
      const [kept] = await client.panels.create([
        { name: `panel-${id.create()}`, project: scope },
        { name: `panel-${id.create()}`, project: scope },
      ]);
      const outsider = await createPanel();
      const retrieved = await client.panels.retrieve({
        projects: [scope],
        key: [kept.key, outsider.key],
      });
      expect(retrieved.map((p) => p.key)).toEqual([kept.key]);
    });
  });

  describe("optimistic rename", () => {
    const WRITE_FAILED = new Error("write failed");
    const fail = () => {
      throw WRITE_FAILED;
    };

    it("should cache the new name before the write commits", async () => {
      const pan = await createPanel();
      let duringWrite: query.Cached<panel.Panel> | undefined;
      await client.panels.rename(pan.key, "renamed", {
        onOptimistic: () => {
          duringWrite = client.panels.getCached(pan.key);
        },
      });
      expect(expectLive(duringWrite).name).toEqual("renamed");
    });

    it("should restore the previous name when the write fails", async () => {
      const pan = await createPanel();
      await expect(
        client.panels.rename(pan.key, "renamed", { onOptimistic: fail }),
      ).rejects.toBe(WRITE_FAILED);
      expect(expectLive(client.panels.getCached(pan.key)).name).toEqual(pan.name);
    });

    it("should restore the ontology resource name when the write fails", async () => {
      const pan = await createPanel();
      const panelID = panel.resourceID(pan.key);
      await client.ontology.retrieve(panelID);
      await expect(
        client.panels.rename(pan.key, "renamed", { onOptimistic: fail }),
      ).rejects.toBe(WRITE_FAILED);
      expect(expectLive(client.ontology.getCached(panelID)).name).toEqual(pan.name);
    });
  });
});
