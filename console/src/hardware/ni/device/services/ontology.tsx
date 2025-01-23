// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { Task } from "@/hardware/ni/task";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

export const ContextMenuItems = ({
  selection: { resources },
}: Ontology.TreeContextMenuProps) => {
  const placeLayout = Layout.usePlacer();
  if (resources.length !== 1) return null;
  const maybeConfigure = () => {
    const first = resources[0];
    if (first.data?.configured === false)
      placeLayout({ ...Device.CONFIGURE_LAYOUT, key: first.id.key });
  };
  const handleCreateDigitalReadTask = () => {
    maybeConfigure();
    placeLayout(Task.DIGITAL_READ_LAYOUT);
  };
  const handleCreateAnalogReadTask = () => {
    maybeConfigure();
    placeLayout(Task.ANALOG_READ_LAYOUT);
  };
  const handleCreateDigitalWriteTask = () => {
    maybeConfigure();
    placeLayout(Task.DIGITAL_WRITE_LAYOUT);
  };
  return (
    <>
      <Common.Task.CreateMenuItem
        itemKey="ni.analogReadTask"
        onClick={handleCreateAnalogReadTask}
      >
        Create Analog Read Task
      </Common.Task.CreateMenuItem>
      <Common.Task.CreateMenuItem
        itemKey="ni.digitalReadTask"
        onClick={handleCreateDigitalReadTask}
      >
        Create Digital Read Task
      </Common.Task.CreateMenuItem>
      <Common.Task.CreateMenuItem
        itemKey="ni.digitalWriteTask"
        onClick={handleCreateDigitalWriteTask}
      >
        Create Digital Write Task
      </Common.Task.CreateMenuItem>
    </>
  );
};
