// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { readPlotChannels, ReadSelectable } from "@/feature/ethercat/task/Read";
import { PREFIX, READ_TYPE } from "@/feature/ethercat/task/types";
import { WriteSelectable } from "@/feature/ethercat/task/Write";
import { type Task } from "@/platform/task";

export const INTEGRATION: Task.Integration = {
  prefix: PREFIX,
  name: "EtherCAT",
  icon: <Icon.Logo.EtherCAT />,
  selectables: [ReadSelectable, WriteSelectable],
  plotChannels: { [READ_TYPE]: readPlotChannels },
};
