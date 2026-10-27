// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { Table } from "@/session/table";
import {
  createSliceStore,
  documentIn,
  DRIFT_STATE,
  inWindow,
} from "@/session/window/testutil";

const storeWith = (slice: Table.SliceState) =>
  createSliceStore({
    name: Table.SLICE_NAME,
    reducer: Table.reducer,
    preloadedState: slice,
    middleware: Table.MIDDLEWARE,
  });

const KEY = "table-1";
const AUX_WINDOW = "aux-window";

describe("Table Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Table.ZERO_SLICE_STATE);
  });

  const exists = (key: string = KEY): boolean =>
    documentIn(store.getState()[Table.SLICE_NAME], key) != null;

  const doc = (key: string = KEY): Table.State =>
    documentIn(store.getState()[Table.SLICE_NAME], key) ?? Table.ZERO_STATE;

  // A window is a viewport: an edit in one window must not reach another window's
  // view of the same document.
  it("should keep each window's view of a document independent", () => {
    store.dispatch(Table.create({ key: KEY }));
    store.dispatch(Table.create({ key: KEY, windowKey: AUX_WINDOW }));
    store.dispatch(
      Table.setEditable({ key: KEY, editable: false, windowKey: AUX_WINDOW }),
    );
    const slice = store.getState()[Table.SLICE_NAME];
    expect(documentIn(slice, KEY, AUX_WINDOW)?.editable).toBe(false);
    expect(documentIn(slice, KEY)?.editable).toBe(true);
  });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      act(() => void store.dispatch(Table.create({ key: KEY })));
      expect(doc()).toEqual(Table.ZERO_STATE);
    });

    it("should create multiple tables independently", () => {
      act(() => {
        store.dispatch(Table.create({ key: "table-1" }));
        store.dispatch(Table.create({ key: "table-2" }));
      });
      expect(exists("table-1")).toBe(true);
      expect(exists("table-2")).toBe(true);
      expect(exists("absent")).toBe(false);
    });

    it("should apply provided fields over the defaults", () => {
      act(() => void store.dispatch(Table.create({ key: KEY, editable: false })));
      expect(doc().editable).toBe(false);
    });

    it("should not overwrite an existing entry", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setEditable({ key: KEY, editable: false }));
        store.dispatch(Table.create({ key: KEY }));
      });
      expect(doc().editable).toBe(false);
    });
  });

  describe("setSelectedCells", () => {
    it("should set the selected cells", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a", "b"] }));
      });
      expect(doc().selectedCells).toEqual(["a", "b"]);
    });

    it("should set the anchor when provided", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(
          Table.setSelectedCells({ key: KEY, cells: ["a", "b"], anchor: "a" }),
        );
      });
      expect(doc().lastSelected).toBe("a");
    });

    it("should leave the anchor untouched when omitted", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"], anchor: "a" }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["b"] }));
      });
      expect(doc().lastSelected).toBe("a");
    });

    it("should lazily create the entry when the key does not exist", () => {
      act(
        () => void store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"] })),
      );
      expect(doc().selectedCells).toEqual(["a"]);
    });
  });

  describe("setEditable", () => {
    it("should toggle editing when no value is provided", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY, editable: true }));
        store.dispatch(Table.setEditable({ key: KEY }));
      });
      expect(doc().editable).toBe(false);
    });

    it("should clear the selection when editing is disabled", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"], anchor: "a" }));
        store.dispatch(Table.setEditable({ key: KEY, editable: false }));
      });
      expect(doc().selectedCells).toEqual([]);
      expect(doc().lastSelected).toBeNull();
    });
  });

  describe("setHideIndicators", () => {
    it("should set the hide indicators flag", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setHideIndicators({ key: KEY, hideIndicators: true }));
      });
      expect(doc().hideIndicators).toBe(true);
    });

    it("should toggle the flag when no value is provided", () => {
      act(() => void store.dispatch(Table.create({ key: KEY })));
      act(() => void store.dispatch(Table.setHideIndicators({ key: KEY })));
      expect(doc().hideIndicators).toBe(true);
      act(() => void store.dispatch(Table.setHideIndicators({ key: KEY })));
      expect(doc().hideIndicators).toBe(false);
    });
  });

  describe("setCentered", () => {
    it("should set the centered flag", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setCentered({ key: KEY, centered: true }));
      });
      expect(doc().centered).toBe(true);
    });

    it("should toggle the flag when no value is provided", () => {
      act(() => void store.dispatch(Table.create({ key: KEY })));
      act(() => void store.dispatch(Table.setCentered({ key: KEY })));
      expect(doc().centered).toBe(true);
      act(() => void store.dispatch(Table.setCentered({ key: KEY })));
      expect(doc().centered).toBe(false);
    });

    it("should lazily create the entry when the key does not exist", () => {
      act(() => void store.dispatch(Table.setCentered({ key: KEY })));
      expect(doc().centered).toBe(true);
    });
  });

  describe("setPropertiesTab", () => {
    it("should leave the properties tab unset by default", () => {
      act(() => void store.dispatch(Table.create({ key: KEY })));
      expect(doc().propertiesTab).toBeUndefined();
    });

    it("should set the properties tab", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setPropertiesTab({ key: KEY, tab: "redline" }));
      });
      expect(doc().propertiesTab).toBe("redline");
    });

    it("should keep the properties tab when the selection changes", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.setPropertiesTab({ key: KEY, tab: "style" }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["a"] }));
        store.dispatch(Table.setSelectedCells({ key: KEY, cells: ["b"] }));
      });
      expect(doc().propertiesTab).toBe("style");
    });
  });

  describe("remove", () => {
    it("should remove a table by key", () => {
      act(() => void store.dispatch(Table.create({ key: KEY })));
      expect(exists()).toBe(true);
      act(() => void store.dispatch(Table.remove({ keys: [KEY] })));
      expect(exists()).toBe(false);
    });

    it("should remove multiple tables at once", () => {
      act(() => {
        store.dispatch(Table.create({ key: "table-1" }));
        store.dispatch(Table.create({ key: "table-2" }));
        store.dispatch(Table.remove({ keys: ["table-1", "table-2"] }));
      });
      expect(exists("table-1")).toBe(false);
      expect(exists("table-2")).toBe(false);
    });

    it("should ignore keys that do not exist", () => {
      act(() => {
        store.dispatch(Table.create({ key: KEY }));
        store.dispatch(Table.remove({ keys: ["absent"] }));
      });
      expect(doc()).toEqual(Table.ZERO_STATE);
    });
  });

  describe("zero state", () => {
    it("should hold the defaults for a new table", () => {
      expect(Table.ZERO_STATE).toEqual({
        editable: true,
        selectedCells: [],
        lastSelected: null,
        hideIndicators: false,
        centered: false,
      });
    });
  });

  describe("sliceStateZ schema", () => {
    it("should default the slice version to 0", () => {
      expect(Table.sliceStateZ.parse({}).version).toBe(0);
      expect(Table.ZERO_SLICE_STATE.version).toBe(0);
    });
  });
});

describe("persistence", () => {
  const persist = (state: Table.State): Table.State | undefined => {
    const stored = {
      ...DRIFT_STATE,
      [Table.SLICE_NAME]: { version: 0 as const, windows: inWindow({ [KEY]: state }) },
    };
    Table.PERSIST_EXCLUDE.forEach((exclude) => exclude(stored));
    return documentIn(stored[Table.SLICE_NAME], KEY);
  };

  // A selection is the state of an edit in progress, so it does not outlive the
  // session that made it.
  it("should clear the selection on the way to disk", () => {
    const purged = persist({
      ...Table.ZERO_STATE,
      selectedCells: ["a", "b"],
      lastSelected: "b",
    });
    expect(purged?.selectedCells).toEqual([]);
    expect(purged?.lastSelected).toBeNull();
  });

  it("should leave the rest of the document alone", () => {
    const purged = persist({
      ...Table.ZERO_STATE,
      selectedCells: ["a"],
      centered: true,
    });
    expect(purged?.centered).toBe(true);
  });
});
