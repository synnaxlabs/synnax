// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Haul } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/platform/css";
import { captureEntries } from "@/platform/import/entries";

// Accepts a drag carrying exactly one OS file or folder.
const canDropFile: Haul.CanDrop = ({ items }) =>
  items.some((item) => item.type === Haul.FILE_TYPE) && items.length === 1;

export interface DropZoneProps extends Omit<Flex.BoxProps, "onDrop" | "onClick"> {
  /** Receives the drop's file-system entries, captured before anything awaits. */
  onDrop: (entries: FileSystemEntry[]) => void;
  /** Opens a picker as the click fallback; when omitted, the zone is not clickable. */
  onClick?: () => void;
}

/**
 * A bordered zone that accepts a single OS file or folder, dropped or clicked. It owns
 * the drag wiring and hover highlight; callers own what happens to the entries.
 */
export const DropZone = ({
  onDrop,
  onClick,
  className,
  children,
  ...rest
}: DropZoneProps): ReactElement => {
  const [draggingOver, setDraggingOver] = useState(false);
  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      if (event == null) return items;
      event.preventDefault();
      setDraggingOver(false);
      onDrop(captureEntries(event.dataTransfer));
      return items;
    },
    [onDrop],
  );
  const handleDragOver = useCallback(() => setDraggingOver(true), []);
  const dropProps = Haul.useDrop({
    type: Haul.FILE_TYPE,
    onDrop: handleDrop,
    canDrop: canDropFile,
    onDragOver: handleDragOver,
  });
  const handleDragLeave = useCallback(() => setDraggingOver(false), []);
  return (
    <Flex.Box
      className={CSS(
        CSS.B("drop-zone"),
        draggingOver && CSS.M("dragging-over"),
        className,
      )}
      align="center"
      justify="center"
      bordered
      rounded="small"
      borderColor={draggingOver ? 9 : 6}
      onDragLeave={handleDragLeave}
      onClick={onClick}
      {...dropProps}
      {...rest}
    >
      {children}
    </Flex.Box>
  );
};
