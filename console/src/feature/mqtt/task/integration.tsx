// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { EdgeSelectable } from "@/feature/mqtt/task/Edge";
import { ReadSelectable } from "@/feature/mqtt/task/Read";
import { PREFIX, READ_SCHEMAS, READ_TYPE } from "@/feature/mqtt/task/types";
import { WriteSelectable } from "@/feature/mqtt/task/Write";
import { Task } from "@/platform/task";

const readPlotChannels: Task.PlotChannels = (config) =>
  Task.enabledChannels(
    READ_SCHEMAS.config
      .parse(config)
      .entries.filter(({ disabled }) => !disabled)
      .flatMap<Task.PlotEntry>((entry) =>
        entry.type === "plain"
          ? entry.messages.flatMap(({ bindings }) =>
              bindings.map(({ channel }) => ({ channel, disabled: false })),
            )
          : [entry],
      ),
  );

export const INTEGRATION: Task.Integration = {
  prefix: PREFIX,
  name: "MQTT",
  icon: <Icon.Logo.MQTT />,
  selectables: [ReadSelectable, WriteSelectable, EdgeSelectable],
  plotChannels: { [READ_TYPE]: readPlotChannels },
};
