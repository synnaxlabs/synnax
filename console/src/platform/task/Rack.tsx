// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Rack as PRack, Task as PTask, Text, Tooltip } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";

import { CSS } from "@/platform/css";
import { useKey } from "@/platform/task/useKey";

export const Rack = () => {
  const { data: rack, retrieve } = PRack.useRetrieveStateful();
  const taskKey = useKey();
  PTask.useRetrieveEffect({
    onChange: ({ data }) => {
      if (data != null && primitive.isNonZero(data.rack)) retrieve({ key: data.rack });
    },
    query: taskKey == null ? undefined : { key: taskKey },
  });
  if (rack == null) return;
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
