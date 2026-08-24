// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Legacy } from "@/session/legacy";

// Reconstructed from the 0.56 store read off this machine before it was overwritten:
// pointer at slot 3, thirteen slices, semver string versions, Local already rekeyed to
// a UUID by the old changeKey repair while Demo kept its literal key.
const BLOB = {
  arc: { arcs: {}, version: "2.0.0" },
  cluster: {
    activeCluster: null,
    clusters: {
      "8edeb842-40e9-4a55-87e9-5f0937b4654a": {
        host: "localhost",
        key: "8edeb842-40e9-4a55-87e9-5f0937b4654a",
        name: "Local",
        password: "seldon",
        port: 9090,
        secure: false,
        username: "synnax",
      },
      DEMO: {
        host: "demo.synnaxlabs.com",
        key: "DEMO",
        name: "Demo",
        password: "seldon",
        port: 9090,
        secure: true,
        username: "synnax",
      },
    },
    version: "3.0.0",
  },
  docs: { location: { heading: "", path: "" }, version: "0.0.0" },
  drift: { windows: {} },
  layout: {
    activeTheme: "synnaxLight",
    altKeyToKey: {},
    colorContext: {
      frequent: {},
      palettes: { frequent: { key: "frequent", name: "Frequent", swatches: [] } },
    },
    keyToAltKey: {},
    layouts: {},
    mosaics: {},
    nav: { main: { drawers: { left: { activeItem: null } } } },
    version: "10.0.0",
  },
  line: { plots: {}, version: "4.0.0" },
  log: { logs: {}, version: "1.0.0" },
  range: { activeRange: null, ranges: [], version: "0.0.0" },
  schematic: { schematics: {}, version: "6.0.0" },
  status: { favorites: [] },
  table: { tables: {}, version: "1.0.0" },
  version: { consoleVersion: "0.56.11", updateNotificationsSilenced: false },
  workspace: { active: null, version: "1.0.0" },
};

const STORE: Record<string, unknown> = {
  "console-version": { version: 3 },
  "console-persisted-state.3": BLOB,
};

const read = async (key: string) => STORE[key] ?? null;

describe("Legacy.migrate over a real 0.56 store", () => {
  it("should carry both Cores across under the keys they already had", async () => {
    const { core } = await Legacy.migrate(read);
    expect(Object.keys(core?.cores ?? {}).sort()).toEqual([
      "8edeb842-40e9-4a55-87e9-5f0937b4654a",
      "DEMO",
    ]);
    expect(core?.cores["8edeb842-40e9-4a55-87e9-5f0937b4654a"]).toMatchObject({
      name: "Local",
      username: "synnax",
      password: "seldon",
      secure: false,
    });
  });

  it("should leave the selection empty when the old store had none", async () => {
    expect((await Legacy.migrate(read)).core?.selected).toBeUndefined();
  });

  it("should map synnaxLight onto the light mode", async () => {
    expect((await Legacy.migrate(read)).theme?.mode).toBe("light");
  });

  it("should carry the recent-color context across", async () => {
    expect((await Legacy.migrate(read)).color?.context).toEqual({ frequent: {} });
  });

  it("should carry no project when the old store had no active workspace", async () => {
    expect(await Legacy.migrate(read)).not.toHaveProperty("project");
  });

  it("should ignore the slices this release does not keep", async () => {
    const migrated = await Legacy.migrate(read);
    expect(Object.keys(migrated).sort()).toEqual(["color", "core", "theme"]);
  });
});
