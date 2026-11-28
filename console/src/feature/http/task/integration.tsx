// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { ReadSelectable, WriteSelectable } from "@/feature/http/task/create";
import { PREFIX, READ_SCHEMAS, READ_TYPE } from "@/feature/http/task/types";
import { Task } from "@/platform/task";

const readPlotChannels: Task.PlotChannels = (config) =>
  READ_SCHEMAS.config
    .parse(config)
    .endpoints.filter(({ disabled }) => !disabled)
    .flatMap(({ icd, messages }) =>
      icd == null ? [] : Task.readChannels(icd, messages),
    );

export const INTEGRATION: Task.Integration = {
  prefix: PREFIX,
  name: "HTTP",
  icon: <Icon.Logo.HTTP />,
  selectables: [ReadSelectable, WriteSelectable],
  plotChannels: { [READ_TYPE]: readPlotChannels },
};
