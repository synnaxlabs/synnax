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

import { createConfigureLayout } from "@/hardware/labjack/device/Configure";
import { configureReadLayout } from "@/hardware/labjack/task/Read";
import { configureWriteLayout } from "@/hardware/labjack/task/Write";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";

interface InitialArgs {
  create: true;
  initialValues: { config: { device: string } };
}

export const ContextMenuItems = ({
  selection: { resources },
}: Ontology.TreeContextMenuProps) => {
  const placer = Layout.usePlacer();
  const first = resources[0];
  const isSingle = resources.length === 1;
  const args: InitialArgs = {
    create: true,
    initialValues: { config: { device: first.id.key } },
  };
  const maybeConfigure = () => {
    if (first.data?.configured === false)
      placer(createConfigureLayout(first.id.key, {}));
  };
  const handleCreateReadTask = () => {
    maybeConfigure();
    placer(configureReadLayout(args));
  };
  const handleCreateWriteTask = () => {
    maybeConfigure();
    placer(configureWriteLayout(args));
  };
  if (!isSingle) return null;
  return (
    <>
      <Menu.Divider />
      <Menu.Item
        startIcon={
          <PIcon.Create>
            <Icon.Task />
          </PIcon.Create>
        }
        itemKey="labjack.readTask"
        onClick={handleCreateReadTask}
      >
        Create Read Task
      </Menu.Item>
      <Menu.Item
        startIcon={
          <PIcon.Create>
            <Icon.Task />
          </PIcon.Create>
        }
        itemKey="labjack.writeTask"
        onClick={handleCreateWriteTask}
      >
        Create Write Task
      </Menu.Item>
    </>
  );
};
