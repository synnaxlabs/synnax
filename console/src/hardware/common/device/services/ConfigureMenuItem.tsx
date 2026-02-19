// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Menu } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

export interface ConfigureMenuItemProps
  extends Pick<Ontology.TreeContextMenuProps, "selection" | "state"> {
  configureLayout: Layout.BaseState;
}

export const ConfigureMenuItem = ({
  configureLayout,
  selection: { ids },
  state: { getResource },
}: ConfigureMenuItemProps) => {
  const placeLayout = Layout.usePlacer();
  const first = getResource(ids[0]);
  if (ids.length !== 1 || first.data?.configured === true) return null;
  const handleClick = () => placeLayout({ ...configureLayout, key: first.id.key });
  return (
    <Menu.Item itemKey="configure" onClick={handleClick}>
      <Icon.Hardware />
      Configure
    </Menu.Item>
  );
};
