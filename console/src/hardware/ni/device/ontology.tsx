// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ZERO_CONFIGURE_LAYOUT } from "@/hardware/ni/device/Configure";
import { createAnalogReadLayout } from "@/hardware/ni/task/AnalogRead";
import { createDigitalReadLayout } from "@/hardware/ni/task/DigitalRead";
import { createDigitalWriteLayout } from "@/hardware/ni/task/DigitalWrite";
import { Task } from "@/hardware/task";
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
      place({ ...ZERO_CONFIGURE_LAYOUT, key: first.id.key });
  };
  const handleCreateDigitalReadTask = () => {
    maybeConfigure();
    place(createDigitalReadLayout(args));
  };
  const handleCreateAnalogReadTask = () => {
    maybeConfigure();
    place(createAnalogReadLayout({ create: true }));
  };
  const handleCreateDigitalWriteTask = () => {
    maybeConfigure();
    place(createDigitalWriteLayout(args));
  };
  if (!isSingle) return null;
  return (
    <>
      <Task.CreateMenuItem
        itemKey="ni.analogReadTask"
        onClick={handleCreateAnalogReadTask}
      >
        Create Analog Read Task
      </Task.CreateMenuItem>
      <Task.CreateMenuItem
        itemKey="ni.digitalReadTask"
        onClick={handleCreateDigitalReadTask}
      >
        Create Digital Read Task
      </Task.CreateMenuItem>
      <Task.CreateMenuItem
        itemKey="ni.digitalWriteTask"
        onClick={handleCreateDigitalWriteTask}
      >
        Create Digital Write Task
      </Task.CreateMenuItem>
    </>
  );
};
