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
  it("should carry every Core across under the key it already had", async () => {
    const [read] = createReader(FULL);
    const { core } = await Legacy.migrate(read);
    expect(Object.keys(core?.cores ?? {}).sort()).toEqual(
      ["DEMO", LOCAL_LEGACY_KEY].sort(),
    );
    expect(core?.cores[LOCAL_LEGACY_KEY]).toMatchObject({
      name: "Local",
      username: "synnax",
      password: "seldon",
    });
  });

  it("should carry the selection across", async () => {
    const [read] = createReader(FULL);
    const { core } = await Legacy.migrate(read);
    expect(core?.selected).toBe(LOCAL_LEGACY_KEY);
  });

  it("should keep Cores that shared an address apart", async () => {
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
    expect(Object.keys(core?.cores ?? {}).sort()).toEqual(["a", "b"]);
  });

  it("should drop a Core whose port cannot parse and keep the rest", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const [read] = createReader({
      ...FULL,
      "console-persisted-state.3": {
        ...BLOB,
        cluster: {
          activeCluster: null,
          clusters: {
            good: { name: "Good", host: "localhost", port: 9090, secure: false },
            bad: { name: "Bad", host: "localhost", port: "not-a-port", secure: false },
          },
        },
      },
    });
    const { core } = await Legacy.migrate(read);
    expect(Object.keys(core?.cores ?? {})).toEqual(["good"]);
    errorSpy.mockRestore();
  });

  it("should carry the recent colors across", async () => {
    const [read] = createReader(FULL);
    expect((await Legacy.migrate(read)).color?.context).toEqual({ frequent: {} });
  });

  it("should read the slot the version pointer names", async () => {
    const [read, spy] = createReader({
      "console-version": { version: 1 },
      "console-persisted-state.1": BLOB,
    });
    expect((await Legacy.migrate(read)).core?.selected).toBe(LOCAL_LEGACY_KEY);
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

  it("should carry the other branches when one fails validation", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const [read] = createReader({
      ...FULL,
      "console-persisted-state.3": { ...BLOB, cluster: { clusters: "corrupt" } },
    });
    const migrated = await Legacy.migrate(read);
    expect(migrated.core).toBeUndefined();
    expect(migrated.color).toBeDefined();
    expect(errorSpy).toHaveBeenCalled();
    errorSpy.mockRestore();
  });

  it("should keep the Cores it can read when one record fails validation", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const [read] = createReader({
      ...FULL,
      "console-persisted-state.3": {
        ...BLOB,
        cluster: {
          activeCluster: null,
          clusters: { ...BLOB.cluster.clusters, broken: { name: "Broken" } },
        },
      },
    });
    const { core } = await Legacy.migrate(read);
    expect(Object.keys(core?.cores ?? {}).sort()).toEqual(
      ["DEMO", LOCAL_LEGACY_KEY].sort(),
    );
    expect(errorSpy).toHaveBeenCalled();
    errorSpy.mockRestore();
  });

  it("should carry a Core whose parameters sat under props", async () => {
    const [read] = createReader({
      ...FULL,
      "console-persisted-state.3": {
        ...BLOB,
        cluster: {
          activeCluster: LOCAL_LEGACY_KEY,
          clusters: {
            [LOCAL_LEGACY_KEY]: {
              key: LOCAL_LEGACY_KEY,
              name: "Local",
              props: {
                host: "localhost",
                port: "9090",
                username: "synnax",
                password: "seldon",
                secure: false,
              },
            },
          },
        },
      },
    });
    const { core } = await Legacy.migrate(read);
    expect(core?.cores[LOCAL_LEGACY_KEY]).toMatchObject({
      name: "Local",
      host: "localhost",
      port: 9090,
      username: "synnax",
      password: "seldon",
    });
    expect(core?.selected).toBe(LOCAL_LEGACY_KEY);
  });

  it("should leave the slices it does not carry at their defaults", async () => {
    const [read] = createReader(FULL);
    const migrated = await Legacy.migrate(read);
    expect(migrated).not.toHaveProperty("range");
    expect(migrated).not.toHaveProperty("status");
    expect(migrated).not.toHaveProperty("panels");
    expect(migrated).not.toHaveProperty("nav");
    // The workspace selection stays behind by design: reselecting is one step. The
    // theme does too: the stored value was only the OS theme at last run.
    expect(migrated).not.toHaveProperty("project");
    expect(migrated).not.toHaveProperty("theme");
  });
});

describe("Legacy migration defaults", () => {
  it("should produce slice states the current schemas accept", async () => {
    const [read] = createReader(FULL);
    const { core, color } = await Legacy.migrate(read);
    expect(() => Core.sliceStateZ.parse(core)).not.toThrow();
    expect(() => Color.sliceStateZ.parse(color)).not.toThrow();
  });
});
