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
  const rackKey = task.getRackKey(taskKey);
  const rack = PRack.use(rackKey);
  if (rack == null) return null;
  return (
    <Tooltip.Dialog>
      <Text.Text level="small" shade={10} weight={450}>
        Task is deployed to {rack.name}
      </Text.Text>
      <Text.WithIcon
        className={CSS.B("rack-name")}
        startIcon={<Icon.Rack />}
        level="small"
        shade={9}
        weight={350}
        style={{ paddingRight: "0.5rem" }}
        ellipsis
      >
        {rack?.name}
      </Text.WithIcon>
    </Tooltip.Dialog>
  );
};
