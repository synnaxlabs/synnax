// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ICD } from "@/session/icd";
import { createSliceStore, documentIn, inWindow } from "@/session/window/testutil";

const storeWith = (slice: ICD.SliceState = ICD.ZERO_SLICE_STATE) =>
  createSliceStore({
    name: ICD.SLICE_NAME,
    reducer: ICD.reducer,
    preloadedState: slice,
    middleware: ICD.MIDDLEWARE,
  });

const KEY = "icd-1";
const AUX_WINDOW = "aux-window";

describe("ICD slice", () => {
  it("should select messages in the dispatching window only", () => {
    const store = storeWith();
    store.dispatch(ICD.setSelected({ key: KEY, selected: ["alarm"] }));
    store.dispatch(
      ICD.setSelected({ key: KEY, selected: ["telemetry"], windowKey: AUX_WINDOW }),
    );
    const slice = store.getState()[ICD.SLICE_NAME];
    expect(documentIn(slice, KEY)?.selected).toEqual(["alarm"]);
    expect(documentIn(slice, KEY, AUX_WINDOW)?.selected).toEqual(["telemetry"]);
  });

  it("should remove an ICD from every window", () => {
    const store = storeWith({
      windows: {
        ...inWindow({ [KEY]: { selected: ["alarm"] }, other: { selected: [] } }),
        ...inWindow({ [KEY]: { selected: ["alarm"] } }, AUX_WINDOW),
      },
    });
    store.dispatch(ICD.remove({ keys: [KEY] }));
    const slice = store.getState()[ICD.SLICE_NAME];
    expect(documentIn(slice, KEY)).toBeUndefined();
    expect(documentIn(slice, KEY, AUX_WINDOW)).toBeUndefined();
    expect(documentIn(slice, "other")).toEqual({ selected: [] });
  });
});
