// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ZERO_CONFIGURE_LAYOUT } from "@/hardware/labjack/device/Configure";
import { createReadLayout } from "@/hardware/labjack/task/Read";
import { createWriteLayout } from "@/hardware/labjack/task/Write";
import { Task } from "@/hardware/task";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

export const ContextMenuItems = ({
  selection: { resources },
}: Ontology.TreeContextMenuProps) => {
  const place = Layout.usePlacer();
  const first = resources[0];
  const isSingle = resources.length === 1;
  const args = { create: true, initialValues: { config: { device: first.id.key } } };
  const maybeConfigure = () => {
    if (first.data?.configured === false)
      place({ ...ZERO_CONFIGURE_LAYOUT, key: first.id.key });
  };
  const handleCreateReadTask = () => {
    maybeConfigure();
    place(createReadLayout(args));
  };
  const handleCreateWriteTask = () => {
    maybeConfigure();
    place(createWriteLayout(args));
  };
  if (!isSingle) return null;
  return (
    <>
      <Task.CreateMenuItem itemKey="labjack.readTask" onClick={handleCreateReadTask}>
        Create Read Task
      </Task.CreateMenuItem>
      <Task.CreateMenuItem itemKey="labjack.writeTask" onClick={handleCreateWriteTask}>
        Create Write Task
      </Task.CreateMenuItem>
    </>
  );
};
