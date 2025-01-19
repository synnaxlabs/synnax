// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CONFIGURE_LAYOUT } from "@/hardware/opc/device/Configure";
import { configureReadLayout } from "@/hardware/opc/task/Read";
import { createWriteLayout } from "@/hardware/opc/task/Write";
import { Task } from "@/hardware/task";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

export const ContextMenuItems = ({
  selection: { resources },
}: Ontology.TreeContextMenuProps): ReactElement | null => {
  const placeLayout = Layout.usePlacer();
  if (resources.length !== 1) return null;
  const key = resources[0].id.key;
  const handleEditConnection = () => placeLayout({ ...CONFIGURE_LAYOUT, key });
  const args = { create: true, initialValues: { config: { device: key } } };
  const handleCreateReadTask = () => placeLayout(configureReadLayout(args));
  const handleCreateWriteTask = () => placeLayout(createWriteLayout(args));
  return (
    <>
      <Menu.Item
        itemKey="opc.connect"
        startIcon={<Icon.Connect />}
        onClick={handleEditConnection}
      >
        Edit Connection
      </Menu.Item>
      <Menu.Divider />
      <Task.CreateMenuItem itemKey="opc.readTask" onClick={handleCreateReadTask}>
        Create a Read Task
      </Task.CreateMenuItem>
      <Task.CreateMenuItem itemKey="opc.writeTask" onClick={handleCreateWriteTask}>
        Create a Write Task
      </Task.CreateMenuItem>
    </>
  );
};
