// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Synnax, Text, Tooltip } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";

import { CSS } from "@/css";

interface RackProps {
  taskKey: task.Key;
}

export const Rack = ({ taskKey }: RackProps) => {
  const client = Synnax.use();
  const rackKey = task.getRackKey(taskKey);
  const rack = useQuery({
    queryKey: ["rack", rackKey, client?.key],
    queryFn: async () => {
      if (rackKey === 0 || client == null) return null;
      return await client.hardware.racks.retrieve(rackKey);
    },
  }).data;
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
        style={{
          paddingRight: "0.5rem",
        }}
        ellipsis
      >
        {rack?.name}
      </Text.WithIcon>
    </Tooltip.Dialog>
  );
};
