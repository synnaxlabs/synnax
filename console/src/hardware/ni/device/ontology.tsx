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

import { configureAnalogReadLayout } from "@/hardware/ni/task/AnalogRead";
import { configureDigitalReadLayout } from "@/hardware/ni/task/DigitalRead";
import { configureDigitalWriteLayout } from "@/hardware/ni/task/DigitalWrite";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => {
  const place = Layout.usePlacer();

  const handleCreateDigitalReadTask = () => place(configureDigitalReadLayout(true));

  const handleCreateAnalogReadTask = () => place(configureAnalogReadLayout(true));

  const handleCreateDigitalWriteTask = () => place(configureDigitalWriteLayout(true));

  return (
    <>
      <Menu.Divider />
      <Menu.Item
        startIcon={<Icon.Task />}
        itemKey="ni.analogReadTask"
        onClick={handleCreateAnalogReadTask}
      >
        New Analog Read Task
      </Menu.Item>
      <Menu.Item
        startIcon={<Icon.Task />}
        itemKey="ni.digitalReadTask"
        onClick={handleCreateDigitalReadTask}
      >
        New Digital Read Task
      </Menu.Item>
      <Menu.Item
        startIcon={<Icon.Task />}
        itemKey="ni.digitalWriteTask"
        onClick={handleCreateDigitalWriteTask}
      >
        New Digital Write Task
      </Menu.Item>
    </>
  );
};
