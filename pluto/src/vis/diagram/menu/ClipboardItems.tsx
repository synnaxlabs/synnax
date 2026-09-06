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

/** Props for {@link ClipboardItems}. */
export interface ClipboardItemsProps {
  cut: () => void;
  copy: () => void;
  paste: () => void;
  hasSelection: boolean;
}

/**
 * Renders the cut, copy, and paste entries of a diagram context menu, wired to the
 * triggers from the diagram's useClipboard. Render inside a {@link Menu}.
 */
export const ClipboardItems = ({
  cut,
  copy,
  paste,
  hasSelection,
}: ClipboardItemsProps): ReactElement => (
  <>
    <Item
      itemKey="cut"
      onClick={cut}
      disabled={!hasSelection}
      triggerIndicator={Triggers.CUT}
    >
      <Icon.Cut />
      Cut
    </Item>
    <Item
      itemKey="copy"
      onClick={copy}
      disabled={!hasSelection}
      triggerIndicator={Triggers.COPY}
    >
      <Icon.Copy />
      Copy
    </Item>
    <Item itemKey="paste" onClick={paste} triggerIndicator={Triggers.PASTE}>
      <Icon.Paste />
      Paste
    </Item>
  </>
);
