// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { kafka } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

import { ReadSelectable } from "@/feature/kafka/task/Read";
import { PREFIX, READ_TYPE } from "@/feature/kafka/task/types";
import { WriteSelectable } from "@/feature/kafka/task/Write";
import { Task } from "@/platform/task";

const readPlotChannels: Task.PlotChannels = (config) =>
  Task.enabledChannels(
    kafka.readConfigZ
      .parse(config)
      .records.filter(({ disabled }) => !disabled)
      .flatMap(({ messages }) =>
        messages.flatMap(({ bindings }) =>
          bindings.map(({ channel }) => ({ channel, disabled: false })),
        ),
      ),
  );

export const INTEGRATION: Task.Integration = {
  prefix: PREFIX,
  name: "Kafka",
  icon: <Icon.Logo.Kafka />,
  selectables: [ReadSelectable, WriteSelectable],
  plotChannels: { [READ_TYPE]: readPlotChannels },
};
