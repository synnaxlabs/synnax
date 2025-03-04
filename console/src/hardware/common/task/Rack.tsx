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
import { Synnax, Text } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";

interface RackProps {
  taskKey: task.Key;
}

export const Rack = ({ taskKey }: RackProps) => {
  const client = Synnax.use();
  const rackKey = task.getRackKey(taskKey);
  const rack = useQuery({
    queryKey: ["rack", rackKey, client?.key],
    queryFn: () => client?.hardware.racks.retrieve(rackKey),
  }).data;
  return rack == null ? null : (
    <Text.WithIcon
      startIcon={<Icon.Rack />}
      level="p"
      shade={7}
      // Right padding aligns this better with the copy buttons.
      style={{ paddingRight: 3 }}
    >
      {rack?.name}
    </Text.WithIcon>
  );
};
