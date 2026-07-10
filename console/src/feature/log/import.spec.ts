// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type ontology, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Access, Flux, type Pluto } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Log } from "@/feature/log";
import { Layout } from "@/platform/layout";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName, waitForPlacedLayout } from "@/testutil";

const V0_ZERO = { key: "", version: "0.0.0", channels: [], remoteCreated: false };
const V1_ZERO = {
  key: "",
  version: "1.0.0",
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  showChannelNames: true,
  showReceiptTimestamp: true,
  toolbar: { activeTab: "channels" },
};
const V2_ZERO = {
  key: "",
  version: "2.0.0",
  toolbar: { activeTab: "channels" },
};

const v1Channel = (channel: number, channelColor = "", precision = -1) => ({
  channel,
  color: channelColor,
  notation: "standard",
  precision,
  alias: "",
  timestamp: { format: "preciseDate", tz: "local" },
});

describe("log state migrations", () => {
  describe("identity", () => {
    it("should leave a v2 state unchanged", () => {
      expect(Log.anyStateZ.parse(V2_ZERO)).toMatchObject(V2_ZERO);
    });

    it("should stamp the latest version on every input", () => {
      [V0_ZERO, V1_ZERO, V2_ZERO].forEach((state) =>
        expect(Log.anyStateZ.parse(state).version).toBe("2.0.0"),
      );
    });
  });

  describe("body -> pendingUpload", () => {
    it("should park a v0 body in pendingUpload with default channel config", () => {
      const migrated = Log.anyStateZ.parse({
        ...V0_ZERO,
        key: "test",
        channels: [1, 2],
        remoteCreated: false,
      });
      expect(migrated.pendingUpload?.channels).toEqual([
        log.channelEntryZ.parse({ channel: 1, color: color.ZERO }),
        log.channelEntryZ.parse({ channel: 2, color: color.ZERO }),
      ]);
    });

    it("should park a v1 body in pendingUpload, preserving display config", () => {
      const migrated = Log.anyStateZ.parse({
        ...V1_ZERO,
        key: "test",
        remoteCreated: false,
        timestampPrecision: 2,
        showChannelNames: false,
        channels: [v1Channel(1, "#ff0000", 3)],
      });
      expect(migrated.pendingUpload).toMatchObject({
        key: "test",
        timestampPrecision: 2,
        hideChannelNames: true,
      });
      expect(migrated.pendingUpload?.channels[0]).toMatchObject({
        channel: 1,
        precision: 3,
        color: color.construct("#ff0000"),
      });
    });

    it("should leave pendingUpload undefined for a remoteCreated log", () => {
      const migrated = Log.anyStateZ.parse({
        ...V1_ZERO,
        key: "test",
        remoteCreated: true,
        channels: [v1Channel(1)],
      });
      expect(migrated.pendingUpload).toBeUndefined();
    });

    it("should preserve the toolbar tab through migration", () => {
      const migrated = Log.anyStateZ.parse({
        ...V1_ZERO,
        key: "test",
        toolbar: { activeTab: "properties" },
      });
      expect(migrated.toolbar.activeTab).toBe("properties");
    });
  });

  describe("v1 color migration into pendingUpload", () => {
    const migratedColor = (channelColor: string) =>
      Log.anyStateZ.parse({
        ...V1_ZERO,
        key: "test",
        remoteCreated: false,
        channels: [v1Channel(1, channelColor)],
      }).pendingUpload?.channels[0].color;

    it("should convert an empty-string color to color.ZERO", () => {
      expect(migratedColor("")).toEqual(color.ZERO);
    });

    it("should convert a hex string to a real color", () => {
      expect(migratedColor("#ff0000")).toEqual(color.construct("#ff0000"));
    });

    it("should fall back to color.ZERO for an unparseable color", () => {
      expect(migratedColor("not-a-color")).toEqual(color.ZERO);
    });
  });

  describe("anyStateZ", () => {
    it("should parse a v2 state as-is", () => {
      expect(Log.anyStateZ.parse(V2_ZERO).version).toBe("2.0.0");
    });

    it("should parse and migrate a v0 state", () => {
      expect(Log.anyStateZ.parse(V0_ZERO).version).toBe("2.0.0");
    });

    it("should parse a persisted v1 log export into a pendingUpload", () => {
      const result = Log.anyStateZ.parse({
        key: "424ef02f-6ec3-4af6-bdd2-242964a747d8",
        version: "1.0.0",
        channels: [
          {
            channel: 1048581,
            color: "",
            notation: "standard",
            precision: -1,
            alias: "",
          },
        ],
        remoteCreated: false,
        timestampPrecision: 0,
        showChannelNames: true,
        showReceiptTimestamp: true,
        toolbar: { activeTab: "channels" },
        type: "log",
      });
      expect(result.version).toBe("2.0.0");
      expect(result.pendingUpload?.channels[0].color).toEqual(color.ZERO);
      expect(result.pendingUpload?.channels[0].timestamp).toEqual({
        format: "preciseDate",
        tz: "local",
      });
    });
  });
});

describe("ingest", () => {
  it("should create the log on the cluster and place its layout", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const { wrapper, store } = await createConsoleWrapper({ client });
    const { result } = renderHook(
      () => ({
        placer: Layout.usePlacer(),
        fluxStore: Flux.useStore<Pluto.FluxStore>(),
        granted: Access.useUpdateGranted(project.TYPE_ONTOLOGY_ID),
      }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.granted).toBe(true));
    const name = uniqueName("imported_log");
    let id: ontology.ID | undefined | void;
    await act(async () => {
      id = await Log.ingest(
        { ...V1_ZERO, key: "imported" },
        {
          layout: { name, type: "log", key: "imported" },
          placeLayout: result.current.placer,
          store: result.current.fluxStore,
          client,
          projectKey: proj.key,
        },
      );
    });
    if (id == null) throw new Error("ingest returned no id");
    const created = await client.logs.retrieve({ key: id.key });
    expect(created.name).toBe(name);
    const placedKey = await waitForPlacedLayout(store, "log");
    expect(placedKey).toBe(id.key);
    expect(Session.Layout.select(store.getState(), placedKey)?.name).toBe(name);
  });
});
