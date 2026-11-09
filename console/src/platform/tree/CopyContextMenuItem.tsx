// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";

import { type ContextMenuProps } from "@/platform/tree/types";
import { useCopy } from "@/platform/tree/useCopy";
import { useResources } from "@/platform/tree/useResource";

export interface CopyContextMenuItemProps extends Pick<
  ContextMenuProps,
  "selection" | "state"
> {}

/** Copies the selected resources, as Cmd+C does in the tree. */
export const CopyContextMenuItem = ({
  selection: { ids },
  state,
}: CopyContextMenuItemProps): ReactElement | null => {
  const resources = useResources(state, ids);
  const { toClipboard } = useCopy();
  if (resources.length === 0) return null;
  return (
    <Menu.Item
      itemKey="copy"
      triggerIndicator={Triggers.COPY}
      onClick={() => toClipboard(resources)}
    >
      <Icon.Copy />
      Copy
    </Menu.Item>
  );
};
