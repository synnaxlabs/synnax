// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { AnalogReadSelectable } from "@/feature/ni/task/AnalogRead";
import { AnalogWriteSelectable } from "@/feature/ni/task/AnalogWrite";
import { CounterReadSelectable } from "@/feature/ni/task/CounterRead";
import { DigitalReadSelectable } from "@/feature/ni/task/DigitalRead";
import { DigitalWriteSelectable } from "@/feature/ni/task/DigitalWrite";
import { PREFIX } from "@/feature/ni/task/types";
import { type Task } from "@/platform/task";

export const INTEGRATION: Task.Integration = {
  prefix: PREFIX,
  name: "NI",
  icon: <Icon.Logo.NI />,
  selectables: [
    AnalogReadSelectable,
    AnalogWriteSelectable,
    CounterReadSelectable,
    DigitalReadSelectable,
    DigitalWriteSelectable,
  ],
};
