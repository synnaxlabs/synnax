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

import { configureReadLayout } from "@/hardware/opc/task/ReadTask";
import { Layout } from "@/layout";

export const ContextMenuItems = () => {
  const place = Layout.usePlacer();
  const createReadTask = () => place(configureReadLayout(true));
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
        onClick={createReadTask}
      >
        Create a Read Task
      </Menu.Item>
    </>
  );
};
