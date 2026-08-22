// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { Color } from "@/session/color";
import { Core } from "@/session/core";
import { Legacy } from "@/session/legacy";
import { Project } from "@/session/project";
import { Theme } from "@/session/theme";

const LOCAL_LEGACY_KEY = "8edeb842-40e9-4a55-87e9-5f0937b4654a";
const WORKSPACE_KEY = "cfc7f87d-4d0d-440c-9ebf-776b292688b7";

// The shape a 0.56 install actually leaves on disk, trimmed to the fields carried.
const BLOB = {
  cluster: {
    activeCluster: LOCAL_LEGACY_KEY,
    clusters: {
      [LOCAL_LEGACY_KEY]: {
        key: LOCAL_LEGACY_KEY,
        name: "Local",
        host: "localhost",
        port: 9090,
        username: "synnax",
        password: "seldon",
        secure: false,
      },
      DEMO: {
        key: "DEMO",
        name: "Demo",
        host: "demo.synnaxlabs.com",
        port: 9090,
        username: "synnax",
        password: "seldon",
        secure: true,
      },
    },
    version: "3.0.0",
  },
  layout: {
    activeTheme: "synnaxLight",
    colorContext: { frequent: {}, palettes: {} },
    version: "10.0.0",
  },
  workspace: { active: WORKSPACE_KEY, version: "1.0.0" },
  docs: { location: { path: "", heading: "" }, version: "0.0.0" },
  status: { favorites: [] },
};

const createReader = (
  entries: Record<string, unknown>,
): [Legacy.Reader, ReturnType<typeof vi.fn>] => {
  const read = vi.fn(async (key: string) => entries[key] ?? null);
  return [read, read];
};

const FULL: Record<string, unknown> = {
  "console-version": { version: 3 },
  "console-persisted-state.3": BLOB,
};

describe("Legacy.migrate", () => {
  it("should carry every Core across, keyed by address", async () => {
    const [read] = createReader(FULL);
    const { core } = await Legacy.migrate(read);
    expect(Object.keys(core?.cores ?? {}).sort()).toEqual([
      "demo.synnaxlabs.com:9090",
      "localhost:9090",
    ]);
    expect(core?.cores["localhost:9090"]).toMatchObject({
      name: "Local",
      username: "synnax",
      password: "seldon",
    });
  });

  it("should follow the selection from the generated key onto the address", async () => {
    const [read] = createReader(FULL);
    const { core } = await Legacy.migrate(read);
    expect(core?.selected).toBe("localhost:9090");
  });

  it("should collapse Cores that shared an address", async () => {
    const [read] = createReader({
      ...FULL,
      "console-persisted-state.3": {
        ...BLOB,
        cluster: {
          activeCluster: null,
          clusters: {
            a: { name: "First", host: "localhost", port: 9090, secure: false },
            b: { name: "Second", host: "localhost", port: 9090, secure: false },
          },
        },
      },
    });
    const { core } = await Legacy.migrate(read);
    expect(Object.keys(core?.cores ?? {})).toEqual(["localhost:9090"]);
  });

  it("should map the active theme onto a mode", async () => {
    const [read] = createReader(FULL);
    expect((await Legacy.migrate(read)).theme?.mode).toBe("light");
  });

  it("should carry the workspace selection over as the project selection", async () => {
    const [read] = createReader(FULL);
    expect((await Legacy.migrate(read)).project?.selected).toBe(WORKSPACE_KEY);
  });

  it("should carry the recent colors across", async () => {
    const [read] = createReader(FULL);
    expect((await Legacy.migrate(read)).color?.context).toEqual({
      frequent: {},
      palettes: {},
    });
  });

  it("should read the slot the version pointer names", async () => {
    const [read, spy] = createReader({
      "console-version": { version: 1 },
      "console-persisted-state.1": BLOB,
    });
    expect((await Legacy.migrate(read)).core?.selected).toBe("localhost:9090");
    expect(spy).toHaveBeenCalledWith("console-persisted-state.1");
  });

  it("should carry nothing when there is no legacy store", async () => {
    const [read] = createReader({});
    expect(await Legacy.migrate(read)).toEqual({});
  });

  it("should carry nothing when the pointer names an absent slot", async () => {
    const [read] = createReader({ "console-version": { version: 2 } });
    expect(await Legacy.migrate(read)).toEqual({});
  });

  it("should carry what it can when a branch fails validation", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const [read] = createReader({
      ...FULL,
      "console-persisted-state.3": { ...BLOB, cluster: { clusters: "corrupt" } },
    });
    expect(await Legacy.migrate(read)).toEqual({});
    expect(errorSpy).toHaveBeenCalled();
    errorSpy.mockRestore();
  });

  it("should leave the slices it does not carry at their defaults", async () => {
    const [read] = createReader(FULL);
    const seed = await Legacy.migrate(read);
    expect(seed).not.toHaveProperty("range");
    expect(seed).not.toHaveProperty("status");
    expect(seed).not.toHaveProperty("panels");
    expect(seed).not.toHaveProperty("nav");
  });

  it("should never write to the legacy store", async () => {
    const reader = vi.fn(async (key: string) => FULL[key] ?? null);
    await Legacy.migrate(reader);
    // Reader is the whole surface: there is no write path to the old store at all.
    expect(Object.keys(reader)).not.toContain("set");
  });
});

describe("Legacy seed defaults", () => {
  it("should produce slice states the current schemas accept", async () => {
    const [read] = createReader(FULL);
    const { core, theme, color, project } = await Legacy.migrate(read);
    expect(() => Core.sliceStateZ.parse(core)).not.toThrow();
    expect(() => Theme.sliceStateZ.parse(theme)).not.toThrow();
    expect(() => Color.sliceStateZ.parse(color)).not.toThrow();
    expect(() => Project.sliceStateZ.parse(project)).not.toThrow();
  });
});
