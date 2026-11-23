// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement, type ReactNode, useState } from "react";
import { afterEach, assert, describe, expect, it, vi } from "vitest";
import { page } from "vitest/browser";

import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { renderThemed } from "@/styletest";
import { mockBoundingClientRect } from "@/testutil";
import { Triggers } from "@/triggers";

const ITEMS: Haul.Item[] = [{ type: "cat", key: "whiskers" }];

const DRAG = { dataTransfer: new DataTransfer() };

const layer = (): HTMLElement => {
  const el = document.querySelector(`.${CSS.BE("haul", "layer")}`);
  assert(el instanceof HTMLElement);
  return el;
};

interface Started {
  /** Whether the preview was the topmost element at its center. */
  top: boolean;
  /** Whether the preview layer was open in the top layer. */
  open: boolean;
}

/** A source that records how its preview showed when the drag started. */
const Source = ({ onStart }: { onStart: (s: Started) => void }): ReactElement => {
  const { startDrag } = Haul.useDrag({ type: "item" });
  return (
    <div
      draggable
      style={{ width: 400, height: 400 }}
      onDragStart={(event) => {
        startDrag({ event, items: [], preview: <p>Preview</p>, anchor: "corner" });
        const preview = page.getByText("Preview").element();
        // The layer lets the pointer through, so a hit test needs it to take it.
        layer().style.pointerEvents = "auto";
        const { left, top, width, height } = preview.getBoundingClientRect();
        const hit = document.elementFromPoint(left + width / 2, top + height / 2);
        onStart({
          top: hit != null && preview.contains(hit),
          open: layer().matches(":popover-open"),
        });
      }}
    >
      Drag
    </div>
  );
};

const Dropdown = ({ onStart }: { onStart: (s: Started) => void }): ReactElement => {
  const [visible, setVisible] = useState(true);
  return (
    <Dialog.Frame visible={visible} onVisibleChange={setVisible}>
      <Dialog.Trigger>Menu</Dialog.Trigger>
      <Dialog.Dialog>
        <Source onStart={onStart} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

const drag = async (
  content: (onStart: (s: Started) => void) => ReactNode,
): Promise<Started> => {
  const started: Started[] = [];
  await renderThemed(
    <Triggers.Provider>
      <Haul.Provider>{content((s) => started.push(s))}</Haul.Provider>
    </Triggers.Provider>,
  );
  const source = page.getByText("Drag");
  await expect.element(source).toBeVisible();
  fireEvent.dragStart(source.element(), { dataTransfer: new DataTransfer() });
  expect(started).toHaveLength(1);
  return started[0];
};

describe("Haul", () => {
  it("should show a drag preview over the dropdown it came from", async () => {
    const started = await drag((onStart) => <Dropdown onStart={onStart} />);
    expect(started).toEqual({ top: true, open: true });
  });

  // The page outside a modal is inert, so a hit test cannot reach the layer there.
  it("should show a drag preview in the top layer over a modal", async () => {
    const started = await drag((onStart) => (
      <Dialog.Frame variant="modal" visible>
        <Dialog.Dialog>
          <Dropdown onStart={onStart} />
        </Dialog.Dialog>
      </Dialog.Frame>
    ));
    expect(started.open).toBe(true);
  });

  describe("preview", () => {
    const PICTURE = "picture of a cat";

    interface PreviewSourceProps {
      anchor?: Haul.PreviewAnchor;
    }

    const PreviewSource = ({ anchor }: PreviewSourceProps): ReactElement => {
      const { startDrag, onDragEnd } = Haul.useDrag({
        type: "test-source",
        key: "source",
      });
      return (
        <button
          data-testid="source"
          onDragStart={(event) =>
            startDrag({ event, items: ITEMS, preview: <span>{PICTURE}</span>, anchor })
          }
          onDragEnd={onDragEnd}
        />
      );
    };

    const spyOnDragImage = () => vi.spyOn(DataTransfer.prototype, "setDragImage");

    afterEach(() => vi.restoreAllMocks());

    it("should picture the preview at dragstart", () => {
      const setDragImage = spyOnDragImage();
      render(
        <Haul.Provider>
          <PreviewSource />
        </Haul.Provider>,
      );
      fireEvent.dragStart(screen.getByTestId("source"), DRAG);
      expect(layer().textContent).toBe(PICTURE);
      expect(setDragImage).toHaveBeenCalledWith(layer(), 4, 8);
    });

    it("should clear the preview once the browser has its picture", async () => {
      render(
        <Haul.Provider>
          <PreviewSource />
        </Haul.Provider>,
      );
      fireEvent.dragStart(screen.getByTestId("source"), DRAG);
      await waitFor(() => expect(layer().childElementCount).toBe(0));
    });

    it("should put the top-left corner of a corner preview on the pointer", () => {
      const setDragImage = spyOnDragImage();
      render(
        <Haul.Provider>
          <PreviewSource anchor="corner" />
        </Haul.Provider>,
      );
      fireEvent.dragStart(screen.getByTestId("source"), DRAG);
      expect(setDragImage).toHaveBeenCalledWith(layer(), 18, 18);
    });

    it("should hold a grab preview at the point the drag pressed", () => {
      const setDragImage = spyOnDragImage();
      render(
        <Haul.Provider>
          <PreviewSource anchor="grab" />
        </Haul.Provider>,
      );
      const source = screen.getByTestId("source");
      source.getBoundingClientRect = mockBoundingClientRect(50, 100, 80, 24);
      fireEvent.dragStart(source, { ...DRAG, clientX: 130, clientY: 62 });
      expect(setDragImage).toHaveBeenCalledWith(layer(), 48, 30);
    });

    it("should dim the source until the drag ends", () => {
      render(
        <Haul.Provider>
          <PreviewSource />
        </Haul.Provider>,
      );
      const source = screen.getByTestId("source");
      fireEvent.dragStart(source, DRAG);
      expect(source.hasAttribute("data-haul-source")).toBe(true);
      fireEvent.dragEnd(source, DRAG);
      expect(source.hasAttribute("data-haul-source")).toBe(false);
    });

    interface TreeItemProps {
      name: string;
      selected: boolean;
    }

    const TreeItem = ({ name, selected }: TreeItemProps): ReactElement => {
      const { startDrag, onDragEnd } = Haul.useDrag({ type: "test-source", key: name });
      return (
        <div
          role="treeitem"
          aria-label={name}
          aria-selected={selected}
          onDragStart={(event) => startDrag({ event, items: ITEMS, preview: name })}
          onDragEnd={onDragEnd}
        />
      );
    };

    const renderTree = (): void => {
      render(
        <Haul.Provider>
          <div role="tree">
            <TreeItem name="a" selected />
            <TreeItem name="b" selected />
            <TreeItem name="c" selected={false} />
          </div>
        </Haul.Provider>,
      );
    };

    const dimmed = (): string[] =>
      screen
        .getAllByRole("treeitem")
        .filter((item) => item.hasAttribute("data-haul-source"))
        .map((item) => item.getAttribute("aria-label") ?? "");

    it("should dim every selected item when a selected tree item drags", () => {
      renderTree();
      const a = screen.getByRole("treeitem", { name: "a" });
      fireEvent.dragStart(a, DRAG);
      expect(dimmed()).toEqual(["a", "b"]);
      fireEvent.dragEnd(a, DRAG);
      expect(dimmed()).toEqual([]);
    });

    it("should dim only the tree item that drags when it is not selected", () => {
      renderTree();
      fireEvent.dragStart(screen.getByRole("treeitem", { name: "c" }), DRAG);
      expect(dimmed()).toEqual(["c"]);
    });
  });
});
