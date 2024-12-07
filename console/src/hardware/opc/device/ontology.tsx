// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Icon as PIcon, Menu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { createConfigureLayout } from "@/hardware/opc/device/Configure";
import { configureReadLayout } from "@/hardware/opc/task/Read";
import { createWriteLayout } from "@/hardware/opc/task/Write";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

interface Args {
  create: boolean;
  initialValues: { config: { device: string } };
}

export const ContextMenuItems = ({
  selection: { resources },
}: Ontology.TreeContextMenuProps): ReactElement | null => {
  const first = resources[0];
  const isSingle = resources.length === 1;
  const place = Layout.usePlacer();
  const initialArgs: Args = {
    create: true,
    initialValues: { config: { device: first.id.key } },
  };
  const createReadTask = () => place(configureReadLayout(initialArgs));
  const createWriteTask = () => place(createWriteLayout(initialArgs));
  const createConfigure = () => place(createConfigureLayout(first.id.key));
  if (!isSingle) return null;
  return (
    <>
      <Menu.Item
        itemKey="opc.connect"
        startIcon={<Icon.Connect />}
        onClick={createConfigure}
      >
        Edit Connection
      </Menu.Item>
      <Menu.Divider />
      <Menu.Item
        startIcon={
          <PIcon.Create>
            <Icon.Task />
          </PIcon.Create>
        }
        itemKey="opc.readTask"
        onClick={createReadTask}
      >
        Create a Read Task
      </Menu.Item>
      <Menu.Item
        startIcon={
          <PIcon.Create>
            <Icon.Task />
          </PIcon.Create>
        }
        itemKey="opc.writeTask"
        onClick={createWriteTask}
      >
        Create a Write Task
      </Menu.Item>
    </>
  );
};
