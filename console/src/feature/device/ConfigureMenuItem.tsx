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

import { type Ontology } from "@/feature/ontology";

export interface ConfigureMenuItemProps extends Pick<
  Ontology.TreeContextMenuProps,
  "selection" | "state"
> {
  onConfigure: (deviceKey: device.Key) => void;
}

export const ConfigureMenuItem = ({
  onConfigure,
  selection: { ids },
  state: { getResource },
}: ConfigureMenuItemProps) => {
  const first = getResource(ids[0]);
  const hasUpdatePermission = Access.useUpdateGranted(device.ontologyID(ids[0].key));
  if (ids.length !== 1 || first.data?.configured === true || !hasUpdatePermission)
    return null;
  const handleClick = () => onConfigure(first.id.key);
  return (
    <Menu.Item itemKey="configure" onClick={handleClick}>
      <Icon.Hardware />
      Configure
    </Menu.Item>
  );
};
