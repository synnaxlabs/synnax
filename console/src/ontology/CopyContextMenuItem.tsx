// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ContextMenu, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { type TreeContextMenuProps } from "@/ontology/service";

export interface CopyContextMenuItemProps
  extends TreeContextMenuProps,
    Pick<ContextMenu.ItemProps, "showBottomDivider"> {}

export const CopyContextMenuItem = ({
  showBottomDivider,
  ...rest
}: CopyContextMenuItemProps): ReactElement | null => {
  const copy = useCopyToClipboard();
  const {
    selection: { ids },
    state: { getResource },
  } = rest;
  if (ids.length !== 1) return null;
  const id = ids[0];
  const { data, name } = getResource(id);
  const handleClick = () => copy(JSON.stringify(data), `data for ${name}`);
  return (
    <ContextMenu.Item
      size="small"
      onClick={handleClick}
      showBottomDivider={showBottomDivider}
    >
      <Icon.Copy />
      Copy properties
    </ContextMenu.Item>
  );
};
