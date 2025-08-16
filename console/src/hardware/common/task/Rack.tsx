// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon, Rack as PRack, Text, Tooltip } from "@synnaxlabs/pluto";

import { CSS } from "@/css";

interface RackProps {
  taskKey: task.Key;
}

export const Rack = ({ taskKey }: RackProps) => {
  const rackKey = task.rackKey(taskKey);
  const rack = PRack.use(rackKey);
  if (rack == null) return null;
  return (
    <Tooltip.Dialog>
      <Text.Text level="small" color={10} weight={450}>
        Task is deployed to {rack.name}
      </Text.Text>
      <Text.Text className={CSS.B("rack-name")} level="small" color={9} weight={350}>
        <Icon.Rack />
        {rack?.name}
      </Text.Text>
    </Tooltip.Dialog>
  );
};
