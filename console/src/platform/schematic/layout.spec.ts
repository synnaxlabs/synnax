// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { type Layout } from "@/platform/layout";
import { Schematic } from "@/platform/schematic";
import { Session } from "@/session";

const invoke = (initial?: Schematic.CreateArg) => {
  const dispatch = vi.fn();
  const props: Layout.CreatorProps = {
    dispatch,
    store: {} as Session.Store,
    windowKey: "main",
  };
  const layout = Schematic.create(initial)(props);
  return { dispatch, layout };
};

describe("schematic layout create", () => {
  it("should return a layout with the schematic defaults", () => {
    const { layout } = invoke();
    expect(layout.type).toBe(Schematic.LAYOUT_TYPE);
    expect(layout.name).toBe("Schematic");
    expect(layout.location).toBe("mosaic");
    expect(layout.icon).toBe("Schematic");
    expect(layout.window).toEqual({ navTop: true, showTitle: true });
  });

  it("should dispatch an internalCreate for the layout key", () => {
    const { dispatch, layout } = invoke();
    expect(layout.key).toBeDefined();
    expect(dispatch).toHaveBeenCalledWith(
      Session.Schematic.internalCreate({ key: layout.key as string }),
    );
  });

  it("should pass through a custom name, location, and tab", () => {
    const tab = { tabKey: "t", mosaicKey: 1 };
    const { layout } = invoke({ name: "My Schematic", location: "window", tab });
    expect(layout.name).toBe("My Schematic");
    expect(layout.location).toBe("window");
    expect(layout.tab).toEqual(tab);
  });

  it("should preserve a valid schematic key", () => {
    const key = uuid.create();
    const { dispatch, layout } = invoke({ key });
    expect(layout.key).toBe(key);
    expect(dispatch).toHaveBeenCalledWith(Session.Schematic.internalCreate({ key }));
  });

  it("should generate a valid key when none is provided", () => {
    const { layout } = invoke();
    expect(() => schematic.keyZ.parse(layout.key)).not.toThrow();
  });

  it("should generate a fresh key when given an invalid one", () => {
    const { layout } = invoke({ key: "not-a-uuid" });
    expect(layout.key).not.toBe("not-a-uuid");
    expect(() => schematic.keyZ.parse(layout.key)).not.toThrow();
  });

  it("should dispatch the same key it returns for the layout", () => {
    const { dispatch, layout } = invoke({ key: "not-a-uuid" });
    const action = dispatch.mock.calls[0][0] as ReturnType<
      typeof Session.Schematic.internalCreate
    >;
    expect(action.payload.key).toBe(layout.key);
  });

  it("should forward the editable flag to the created slice entry", () => {
    const { dispatch } = invoke({ editable: true });
    const action = dispatch.mock.calls[0][0] as ReturnType<
      typeof Session.Schematic.internalCreate
    >;
    expect(action.payload.editable).toBe(true);
  });
});
