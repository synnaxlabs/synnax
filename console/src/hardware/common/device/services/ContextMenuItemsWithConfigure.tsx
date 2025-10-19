// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ContextMenu, Icon } from "@synnaxlabs/pluto";

import {
  ContextMenuItems,
  type ContextMenuItemsProps,
} from "@/hardware/common/device/services/ContextMenuItems";
import { Layout } from "@/layout";

export interface ContextMenuItemsWithConfigureProps extends ContextMenuItemsProps {
  itemKey: string;
}

export const ContextMenuItemsWithConfigure = ({
  itemKey,
  ...props
}: ContextMenuItemsWithConfigureProps) => {
  const {
    configureLayout,
    selection: { ids },
  } = props;
  const placeLayout = Layout.usePlacer();
  if (ids.length !== 1) return null;
  const handleEditConnection = () =>
    placeLayout({ ...configureLayout, key: ids[0].key });
  return (
    <ContextMenuItems {...props}>
      <ContextMenu.Item onClick={handleEditConnection} showBottomDivider>
        <Icon.Edit />
        Edit connection
      </ContextMenu.Item>
    </ContextMenuItems>
  );
};
