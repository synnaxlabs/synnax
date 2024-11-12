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

import { createConfigureLayout } from "@/hardware/ni/device/Configure";
import { configureAnalogReadLayout } from "@/hardware/ni/task/AnalogRead";
import { configureDigitalReadLayout } from "@/hardware/ni/task/DigitalRead";
import { configureDigitalWriteLayout } from "@/hardware/ni/task/DigitalWrite";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

interface InitialArgs {
  create: true;
  initialValues: { config: { device: string } };
}

export const ContextMenuItems = ({
  selection: { resources },
}: Ontology.TreeContextMenuProps) => {
  const place = Layout.usePlacer();
  const first = resources[0];
  const isSingle = resources.length === 1;
  const args: InitialArgs = {
    create: true,
    initialValues: { config: { device: first.id.key } },
  };
  const maybeConfigure = () => {
    if (first.data?.configured === false)
      place(createConfigureLayout(first.id.key, {}));
  };
  const handleCreateDigitalReadTask = () => {
    maybeConfigure();
    place(configureDigitalReadLayout(args));
  };
  const handleCreateAnalogReadTask = () => {
    maybeConfigure();
    place(configureAnalogReadLayout({ create: true }));
  };
  const handleCreateDigitalWriteTask = () => {
    maybeConfigure();
    place(configureDigitalWriteLayout(args));
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
        itemKey="ni.analogReadTask"
        onClick={handleCreateAnalogReadTask}
      >
        Create Analog Read Task
      </Menu.Item>
      <Menu.Item
        startIcon={
          <PIcon.Create>
            <Icon.Task />
          </PIcon.Create>
        }
        itemKey="ni.digitalReadTask"
        onClick={handleCreateDigitalReadTask}
      >
        Create Digital Read Task
      </Menu.Item>
      <Menu.Item
        startIcon={
          <PIcon.Create>
            <Icon.Task />
          </PIcon.Create>
        }
        itemKey="ni.digitalWriteTask"
        onClick={handleCreateDigitalWriteTask}
      >
        Create Digital Write Task
      </Menu.Item>
    </>
  );
};
