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
import { useEffect } from "react";

import { CSS } from "@/css";
import { useKey } from "@/hardware/common/task/useKey";

export const Rack = () => {
  const { data: rack, retrieve } = PRack.useRetrieveStateful();
  const taskKey = useKey();
  useEffect(() => {
    if (taskKey != null) retrieve({ key: task.rackKey(taskKey) });
  }, [taskKey]);
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
