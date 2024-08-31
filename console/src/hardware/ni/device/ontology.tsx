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
import { Ontology } from "@/ontology";

export const ContextMenuItems = ({
  selection: { resources },
}: Ontology.TreeContextMenuProps) => {
  const place = Layout.usePlacer();
  const first = resources[0];
  const isSingle = resources.length === 1;
  const handleCreateDigitalReadTask = () => {
    if (first.data?.configured === false)
      place(createConfigureLayout(first.id.key, {}));
    place(
      configureDigitalReadLayout({
        create: true,
        initialValues: { config: { device: first.id.key } },
      }),
    );
  };
  const handleCreateAnalogReadTask = () => {
    if (first.data?.configured === false)
      place(createConfigureLayout(first.id.key, {}));
    place(
      configureAnalogReadLayout({
        create: true,
        initialValues: { config: { device: first.id.key } },
      }),
    );
  };
  const handleCreateDigitalWriteTask = () => {
    if (first.data?.configured === false)
      place(createConfigureLayout(first.id.key, {}));
    place(
      configureDigitalWriteLayout({
        create: true,
        initialValues: { config: { device: first.id.key } },
      }),
    );
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
