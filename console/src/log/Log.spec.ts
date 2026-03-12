// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

const mockInternalCreate = vi.hoisted(() =>
  vi.fn((payload) => ({ type: "log/create", payload })),
);

vi.mock("@/log/slice", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as object),
    internalCreate: mockInternalCreate,
  };
});

// Workspace.createSyncComponent returns a function — stub it so the module loads
vi.mock("@/workspace", () => ({
  Workspace: {
    createSyncComponent: vi.fn(() => vi.fn()),
  },
}));

// Stub all other heavy imports so the module resolves quickly in a unit test
vi.mock("@synnaxlabs/client", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as object),
    log: {
      ...((actual as Record<string, unknown>).log as object),
      keyZ: {
        safeParse: (v: unknown) => ({
          data: typeof v === "string" && v !== "" ? v : undefined,
        }),
      },
      ontologyID: vi.fn((key: string) => ({ type: "log", key })),
      TYPE_ONTOLOGY_ID: { type: "log" },
    },
  };
});

vi.mock("@synnaxlabs/drift/react", () => ({
  useSelectWindowKey: vi.fn(() => "window-1"),
}));

vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as object),
    Access: {
      ...((actual as Record<string, unknown>).Access as object),
      useUpdateGranted: vi.fn(() => true),
      updateGranted: vi.fn(() => true),
    },
    Log: {
      ...((actual as Record<string, unknown>).Log as object),
      useRetrieveObservable: vi.fn(() => null),
    },
    usePrevious: vi.fn((v: unknown) => v),
    telem: {
      streamMultiChannelLog: vi.fn(() => ({})),
      noopLogSourceSpec: {},
    },
  };
});

vi.mock("@/layout", () => ({
  Layout: {
    useSelectRequired: vi.fn(() => ({ name: "Test Log" })),
    selectRequired: vi.fn(() => ({ name: "Test Log" })),
    select: vi.fn(() => ({ name: "Test Log" })),
    Renderer: {},
    rename: vi.fn((p) => ({ type: "layout/rename", payload: p })),
    setNavDrawerVisible: vi.fn((p) => ({
      type: "layout/setNavDrawerVisible",
      payload: p,
    })),
  },
}));

vi.mock("@/log/selectors", () => ({
  select: vi.fn(),
  useSelect: vi.fn(),
  useSelectVersion: vi.fn(() => "0.0.0"),
}));

vi.mock("@/components", () => ({
  EmptyAction: () => null,
}));

vi.mock("@/hooks/useLoadRemote", () => ({
  createLoadRemote: vi.fn(() => vi.fn(() => null)),
}));

vi.mock("@/selector", () => ({
  Selector: {
    Selectable: {},
    Item: () => null,
  },
}));

// ---------------------------------------------------------------------------
// Import module under test
// ---------------------------------------------------------------------------

import { create, LAYOUT_TYPE } from "@/log/Log";
import { ZERO_STATE } from "@/log/slice";

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("log/Log", () => {
  describe("LAYOUT_TYPE", () => {
    it('is the string "log"', () => {
      expect(LAYOUT_TYPE).toBe("log");
    });
  });

  describe("create()", () => {
    it("returns a Layout.Creator function", () => {
      const creator = create();
      expect(typeof creator).toBe("function");
    });

    it("default name is 'Log'", () => {
      const dispatch = vi.fn();
      const result = create()({ dispatch });
      expect(result.name).toBe("Log");
    });

    it("default location is 'mosaic'", () => {
      const dispatch = vi.fn();
      const result = create()({ dispatch });
      expect(result.location).toBe("mosaic");
    });

    it("type is always LAYOUT_TYPE", () => {
      const dispatch = vi.fn();
      const result = create()({ dispatch });
      expect(result.type).toBe(LAYOUT_TYPE);
    });

    it("icon is always 'Log'", () => {
      const dispatch = vi.fn();
      const result = create()({ dispatch });
      expect(result.icon).toBe("Log");
    });

    it("uses provided name", () => {
      const dispatch = vi.fn();
      const result = create({ name: "Custom Name" })({ dispatch });
      expect(result.name).toBe("Custom Name");
    });

    it("uses provided location", () => {
      const dispatch = vi.fn();
      const result = create({ location: "window" })({ dispatch });
      expect(result.location).toBe("window");
    });

    it("uses provided key when it is a valid log key", () => {
      const dispatch = vi.fn();
      const key = "my-provided-key";
      const result = create({ key })({ dispatch });
      expect(result.key).toBe(key);
    });

    it("generates a uuid when key is empty/absent", () => {
      const dispatch = vi.fn();
      const result = create()({ dispatch });
      expect(typeof result.key).toBe("string");
      expect(result.key.length).toBeGreaterThan(0);
    });

    it("windowKey equals the layout key", () => {
      const dispatch = vi.fn();
      const result = create({ key: "wk-test" })({ dispatch });
      expect(result.windowKey).toBe(result.key);
    });

    it("calls dispatch with internalCreate action", () => {
      const dispatch = vi.fn();
      create({ key: "dispatch-test" })({ dispatch });
      expect(dispatch).toHaveBeenCalledOnce();
      const dispatchedAction = dispatch.mock.calls[0][0];
      // internalCreate is the mocked function that returns { type, payload }
      expect(dispatchedAction.type).toBe("log/create");
    });

    it("dispatched action contains the key", () => {
      const dispatch = vi.fn();
      create({ key: "keyed-log" })({ dispatch });
      const dispatchedAction = dispatch.mock.calls[0][0];
      expect(dispatchedAction.payload.key).toBe("keyed-log");
    });

    it("dispatched payload merges ZERO_STATE defaults", () => {
      const dispatch = vi.fn();
      create({ key: "zero-merge" })({ dispatch });
      const dispatchedAction = dispatch.mock.calls[0][0];
      expect(dispatchedAction.payload.channels).toEqual(ZERO_STATE.channels);
      expect(dispatchedAction.payload.remoteCreated).toBe(ZERO_STATE.remoteCreated);
      expect(dispatchedAction.payload.timestampPrecision).toBe(
        ZERO_STATE.timestampPrecision,
      );
    });

    it("forwards window and tab options to the layout object", () => {
      const dispatch = vi.fn();
      const windowOpts = { title: "My Window" };
      const tabOpts = { closable: false };
      const result = create({ key: "opts-test", window: windowOpts, tab: tabOpts })({
        dispatch,
      });
      expect(result.window).toEqual(windowOpts);
      expect(result.tab).toEqual(tabOpts);
    });

    it("name/location/window/tab are NOT included in the dispatched state payload", () => {
      const dispatch = vi.fn();
      create({ name: "Should Not Appear", location: "modal", key: "exclude-test" })({
        dispatch,
      });
      const payload = dispatch.mock.calls[0][0].payload;
      // 'name' and 'location' belong to Layout.BaseState, not log.State
      expect(payload.name).toBeUndefined();
      expect(payload.location).toBeUndefined();
    });
  });
});
