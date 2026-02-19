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

export interface EditConnectionMenuItemProps extends Pick<
  Ontology.TreeContextMenuProps,
  "selection"
> {
  configureLayout: Layout.BaseState;
}

export const EditConnectionMenuItem = ({
  configureLayout,
  selection: { ids },
}: EditConnectionMenuItemProps) => {
  const placeLayout = Layout.usePlacer();
  if (ids.length !== 1) return null;
  const handleClick = () => placeLayout({ ...configureLayout, key: ids[0].key });
  return (
    <Menu.Item itemKey="editConnection" onClick={handleClick}>
      <Icon.Edit />
      Edit connection
    </Menu.Item>
  );
};
