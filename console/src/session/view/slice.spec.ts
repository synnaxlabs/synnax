// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Session } from "@/session";

const { reducer, removeTabs, removeViews, setOpen, ZERO_SLICE_STATE } = Session.View;

const TAB_A = uuid.create();
const TAB_B = uuid.create();
const VIEW_A = uuid.create();
const VIEW_B = uuid.create();

const withOpen = (open: Record<string, string>): Session.View.SliceState => ({
  ...ZERO_SLICE_STATE,
  open,
});

describe("View slice", () => {
  it("should start with no tab showing a saved view", () => {
    expect(Session.View.sliceStateZ.parse({})).toEqual({ version: 0, open: {} });
  });

  it("should keep the saved view a tab opens", () => {
    const state = reducer(ZERO_SLICE_STATE, setOpen({ tab: TAB_A, view: VIEW_A }));
    expect(state.open).toEqual({ [TAB_A]: VIEW_A });
  });

  it("should drop the entry of a tab that goes back to its static view", () => {
    const state = reducer(
      withOpen({ [TAB_A]: VIEW_A, [TAB_B]: VIEW_B }),
      setOpen({ tab: TAB_A, view: null }),
    );
    expect(state.open).toEqual({ [TAB_B]: VIEW_B });
  });

  it("should drop every tab that shows a removed view", () => {
    const state = reducer(
      withOpen({ [TAB_A]: VIEW_A, [TAB_B]: VIEW_A, [uuid.create()]: VIEW_B }),
      removeViews({ keys: [VIEW_A] }),
    );
    expect(Object.values(state.open)).toEqual([VIEW_B]);
  });

  it("should drop the entries of removed tabs", () => {
    const state = reducer(
      withOpen({ [TAB_A]: VIEW_A, [TAB_B]: VIEW_B }),
      removeTabs({ keys: [TAB_A] }),
    );
    expect(state.open).toEqual({ [TAB_B]: VIEW_B });
  });
});
