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

// import { useQueryRack } from "@/whisper/Whisper";

interface RackProps {
  taskKey: task.Key;
}

export const Rack = ({ taskKey }: RackProps) => {
  const client = Synnax.use();
  const rackKey = task.getRackKey(taskKey);
  // const rack = useQueryRack({
  //   queryKey: ["rack", rackKey, client?.key],
  //   retrieve: async ({ client }) => await client.hardware.racks.retrieve(rackKey),
  // });
  // if (rack == null) return null;
  const rack = {};
  return (
    <Tooltip.Dialog>
      <Text.Text level="small" shade={8} weight={450}>
        Task is deployed to {rack.name}
      </Text.Text>
      <Text.WithIcon
        startIcon={<Icon.Rack />}
        level="p"
        shade={7}
        style={{ paddingRight: "0.5rem" }}
      >
        {rack?.name}
      </Text.WithIcon>
    </Tooltip.Dialog>
  );
};
