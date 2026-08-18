// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { FS } from "@/platform/fs";
import {
  fakeFileEntry,
  FileDragSource,
  fireFileDrop,
  startFileDrag,
} from "@/platform/fs/testutil";
import { renderWithConsole } from "@/testutil";

const TWO_FILE_DRAG = "start two file drag";

// Puts a two-item file haul in flight, which the zone must reject.
const TwoFileDragSource = (): ReactElement => {
  const { startDrag } = Haul.useDrag({ type: "os-file", key: "two-files" });
  return (
    <button
      onClick={() =>
        startDrag([Haul.FILE, { key: "second-file", type: Haul.FILE_TYPE }])
      }
    >
      {TWO_FILE_DRAG}
    </button>
  );
};
TwoFileDragSource.displayName = "TwoFileDragSource";

interface RenderParams {
  onDrop?: (entries: FileSystemEntry[]) => void;
  onClick?: () => void;
}

const renderZone = async ({ onDrop = vi.fn(), onClick }: RenderParams = {}) => {
  const { container } = await renderWithConsole(
    <Haul.Provider>
      <FileDragSource />
      <TwoFileDragSource />
      <FS.DropZone onDrop={onDrop} onClick={onClick}>
        <span>zone content</span>
      </FS.DropZone>
    </Haul.Provider>,
  );
  const zone = container.querySelector<HTMLElement>(".console-drop-zone");
  if (zone == null) throw new Error("drop zone not rendered");
  return { zone };
};

describe("FS.DropZone", () => {
  it("renders its children and opens the fallback on click", async () => {
    const onClick = vi.fn();
    const { zone } = await renderZone({ onClick });
    expect(screen.getByText("zone content")).toBeTruthy();
    fireEvent.click(zone);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("hands the drop's entries to onDrop", async () => {
    const onDrop = vi.fn();
    const { zone } = await renderZone({ onDrop });
    startFileDrag();
    fireFileDrop(zone, [fakeFileEntry(new File(["{}"], "bundle.zip"))]);
    expect(onDrop).toHaveBeenCalledTimes(1);
    const entries = onDrop.mock.calls[0][0];
    expect(entries).toHaveLength(1);
    expect(entries[0].name).toBe("bundle.zip");
  });

  it("highlights while a file drag hovers and clears when it leaves", async () => {
    const { zone } = await renderZone();
    startFileDrag();
    fireEvent.dragOver(zone);
    expect(zone.className).toContain("dragging-over");
    fireEvent.dragLeave(zone);
    expect(zone.className).not.toContain("dragging-over");
  });

  it("ignores a drag carrying more than one item", async () => {
    const onDrop = vi.fn();
    const { zone } = await renderZone({ onDrop });
    fireEvent.click(screen.getByText(TWO_FILE_DRAG));
    fireEvent.dragOver(zone);
    expect(zone.className).not.toContain("dragging-over");
    fireFileDrop(zone, [fakeFileEntry(new File(["{}"], "a.json"))]);
    expect(onDrop).not.toHaveBeenCalled();
  });
});
