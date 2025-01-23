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

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import { Task } from "@/hardware/opc/task";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

export const ContextMenuItems = ({
  selection: { resources },
}: Ontology.TreeContextMenuProps): ReactElement | null => {
  const placeLayout = Layout.usePlacer();
  if (resources.length !== 1) return null;
  const handleEditConnection = () =>
    placeLayout({ ...Device.CONFIGURE_LAYOUT, key: resources[0].id.key });
  const handleCreateReadTask = () => placeLayout(Task.READ_LAYOUT);
  const handleCreateWriteTask = () => placeLayout(Task.WRITE_LAYOUT);
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
      <Common.Task.CreateMenuItem itemKey="opc.readTask" onClick={handleCreateReadTask}>
        Create a Read Task
      </Common.Task.CreateMenuItem>
      <Common.Task.CreateMenuItem
        itemKey="opc.writeTask"
        onClick={handleCreateWriteTask}
      >
        Create a Write Task
      </Common.Task.CreateMenuItem>
    </>
  );
};
