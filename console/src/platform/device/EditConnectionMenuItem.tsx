// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Access, Icon, Menu } from "@synnaxlabs/pluto";

import { type Tree } from "@/platform/tree";

export interface EditConnectionMenuItemProps extends Pick<
  Tree.ContextMenuProps,
  "selection"
> {
  onConfigure: (deviceKey: device.Key) => void;
}

export const EditConnectionMenuItem = ({
  onConfigure,
  selection: { ids },
}: EditConnectionMenuItemProps) => {
  const first = ids.at(0);
  const hasUpdatePermission = Access.useUpdateGranted(
    first != null ? device.ontologyID(first.key) : device.TYPE_ONTOLOGY_ID,
  );
  if (first == null || ids.length !== 1 || !hasUpdatePermission) return null;
  const handleClick = () => onConfigure(first.key);
  return (
    <Menu.Item itemKey="editConnection" onClick={handleClick}>
      <Icon.Edit />
      Edit connection
    </Menu.Item>
  );
};
