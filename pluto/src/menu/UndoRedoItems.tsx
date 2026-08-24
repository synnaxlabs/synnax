// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@/icon";
import { Item } from "@/menu/Item";
import { Triggers } from "@/triggers";

/** Props for {@link UndoRedoItems}. */
export interface UndoRedoItemsProps {
  undo: () => void;
  redo: () => void;
  canUndo: boolean;
  canRedo: boolean;
}

/**
 * Renders the undo and redo entries of a context menu, each hinting the shortcut its
 * host already binds. Render inside a {@link Menu}.
 */
export const UndoRedoItems = ({
  undo,
  redo,
  canUndo,
  canRedo,
}: UndoRedoItemsProps): ReactElement => (
  <>
    <Item
      itemKey="undo"
      onClick={undo}
      disabled={!canUndo}
      triggerIndicator={Triggers.UNDO}
    >
      <Icon.Undo />
      Undo
    </Item>
    <Item
      itemKey="redo"
      onClick={redo}
      disabled={!canRedo}
      triggerIndicator={Triggers.REDO}
    >
      <Icon.Redo />
      Redo
    </Item>
  </>
);
