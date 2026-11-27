// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/used-by/Popover.css";

import { resource } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Text } from "@synnaxlabs/lyra/text";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { location, state } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { CSS } from "@/platform/css";
import { List, type Selects, type Use } from "@/platform/used-by/List";

const POPOVER_LOCATION = {
  targetCorner: location.BOTTOM_RIGHT,
  dialogCorner: location.TOP_RIGHT,
};

// How long the popover stays open after the pointer leaves, so the pointer can cross
// the gap from the count to the popover.
const HOVER_CLOSE_DELAY = 150;

/** Returns the ID of the popover that lists the uses of the item with the key. */
export const dialogID = (key: string): string => `used-by-${key}`;

type PopoverMode = "hover" | "pinned" | null;

export interface PopoverProps {
  /** The key of the item, which names its popover for {@link dialogID}. */
  itemKey: string;
  /** The kind of the item, such as "secret". */
  kind: string;
  /** The uses of the item, or undefined while they load. */
  uses?: Use[];
  /** The element the popover opens below. */
  anchor: HTMLElement | null;
  /**
   * Selects a use of a resource type in place of its tree item, and closes the popover.
   * Keyed by the type.
   */
  selects?: Selects;
}

/**
 * The count of the resources that use an item. Hovering or clicking it opens a popover
 * that lists them, as does {@link dialogID} through `Dialog.open`.
 */
export const Popover = ({
  itemKey,
  kind,
  uses,
  anchor,
  selects,
}: PopoverProps): ReactElement => {
  const [mode, setMode] = useState<PopoverMode>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  useEffect(() => () => clearTimeout(closeTimer.current), []);
  const cancelClose = () => clearTimeout(closeTimer.current);
  const scheduleClose = () => {
    cancelClose();
    closeTimer.current = setTimeout(
      () => setMode((m) => (m === "hover" ? null : m)),
      HOVER_CLOSE_DELAY,
    );
  };
  const handleEnter = () => {
    cancelClose();
    setMode((m) => m ?? "hover");
  };
  const handleVisibleChange = (arg: state.SetArg<boolean>) =>
    setMode((m) => (state.resolve(arg, m != null) ? "pinned" : null));
  // A resource that both reads and writes the item counts once.
  const count = new Set(uses?.map(({ id }) => resource.idToString(id))).size;
  return (
    <Dialog.Frame
      id={dialogID(itemKey)}
      anchor={anchor}
      visible={mode != null}
      onVisibleChange={handleVisibleChange}
      location={POPOVER_LOCATION}
    >
      {count > 0 && (
        <Button.Button
          variant="text"
          size="small"
          textColor={9}
          onDoubleClick={stopPropagation}
          onPointerEnter={handleEnter}
          onPointerLeave={scheduleClose}
          onClick={() => setMode("pinned")}
        >
          Used by {count}
        </Button.Button>
      )}
      <Dialog.Dialog
        background={1}
        rounded="medium"
        className={CSS.BE("used-by", "popover")}
        onPointerEnter={cancelClose}
        onPointerLeave={mode === "hover" ? scheduleClose : undefined}
        onClick={stopPropagation}
      >
        <Text.Text level="small" color={9} className={CSS.BE("used-by", "label")}>
          Used by
        </Text.Text>
        {uses != null && (
          <List
            kind={kind}
            uses={uses}
            selects={selects}
            onSelect={() => setMode(null)}
          />
        )}
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
