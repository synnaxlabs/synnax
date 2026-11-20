// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent } from "@testing-library/react";
import { type ReactElement, type ReactNode, useState } from "react";
import { assert, describe, expect, it } from "vitest";
import { page } from "vitest/browser";

import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { renderThemed } from "@/styletest";
import { Triggers } from "@/triggers";

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
});
