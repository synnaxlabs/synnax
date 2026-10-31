// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Icon } from "@synnaxlabs/lyra/icon";
import { type Resize } from "@synnaxlabs/lyra/resize";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";

export interface Toolbar extends Pick<Resize.SingleProps, "sizeBounds"> {
  key: string;
  content: ReactElement;
  initialSize?: number;
  icon: Icon.ReactElement;
  tooltip: string;
  trigger: Triggers.Trigger;
  useVisible?: () => boolean;
  /** Shows on the nav icon of the toolbar, such as a {@link Badge}. */
  badge?: ReactElement;
}

/**
 * The keys of the toolbars the subject may open. Calls every toolbar's own visibility
 * hook, so items has to be a module-level constant to keep the hook order stable.
 */
export const useVisibleKeys = (items: Toolbar[]): Set<string> => {
  "use no memo"; // Calls a hook per item, so the item list sets the hook order.
  return new Set(
    // eslint-disable-next-line react-hooks/rules-of-hooks
    items.filter(({ useVisible }) => useVisible?.() ?? true).map(({ key }) => key),
  );
};
