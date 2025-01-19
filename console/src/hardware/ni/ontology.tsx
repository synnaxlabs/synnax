// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@/hardware/ni/device";
import { Task } from "@/hardware/ni/task";
import { Task as CoreTask } from "@/hardware/task";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

export const ContextMenuItems = ({
  selection: { resources },
}: Ontology.TreeContextMenuProps) => {
  const placeLayout = Layout.usePlacer();
  if (resources.length !== 1) return null;
  const first = resources[0];
  const key = first.id.key;
  const maybeConfigure = () => {
    if (first.data?.configured === false)
      placeLayout({ ...Device.CONFIGURE_LAYOUT, key });
  };
  const args = { create: true, initialValues: { config: { device: key } } };
  const handleCreateDigitalReadTask = () => {
    maybeConfigure();
    placeLayout(Task.createDigitalReadLayout(args));
  };
  const handleCreateAnalogReadTask = () => {
    maybeConfigure();
    placeLayout(Task.createAnalogReadLayout({ create: true }));
  };
  const handleCreateDigitalWriteTask = () => {
    maybeConfigure();
    placeLayout(Task.createDigitalWriteLayout(args));
  };
  return (
    <>
      <CoreTask.CreateMenuItem
        itemKey="ni.analogReadTask"
        onClick={handleCreateAnalogReadTask}
      >
        Create Analog Read Task
      </CoreTask.CreateMenuItem>
      <CoreTask.CreateMenuItem
        itemKey="ni.digitalReadTask"
        onClick={handleCreateDigitalReadTask}
      >
        Create Digital Read Task
      </CoreTask.CreateMenuItem>
      <CoreTask.CreateMenuItem
        itemKey="ni.digitalWriteTask"
        onClick={handleCreateDigitalWriteTask}
      >
        Create Digital Write Task
      </CoreTask.CreateMenuItem>
    </>
  );
};
