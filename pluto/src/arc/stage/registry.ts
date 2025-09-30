// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Constant } from "@/arc/stage/constant";
import { Operator } from "@/arc/stage/operator";
import { Select } from "@/arc/stage/select";
import { Source } from "@/arc/stage/source";
import { StableFor } from "@/arc/stage/stable";
import { Status } from "@/arc/stage/status";
import { type Spec } from "@/arc/stage/types/spec";
import { Icon } from "@/icon";

export const REGISTRY: Record<string, Spec<any>> = {
  ...Source.SYMBOLS,
  ...Constant.SYMBOLS,
  ...Select.SYMBOLS,
  ...Status.SYMBOLS,
  ...Operator.SYMBOLS,
  ...StableFor.SYMBOLS,
};

export interface Group {
  key: string;
  name: string;
  Icon: Icon.FC;
  symbols: string[];
}

export const GROUPS: Group[] = [
  {
    key: "basic",
    name: "Basic",
    Icon: Icon.Schematic,
    symbols: [...Object.keys(Constant.SYMBOLS), ...Object.keys(Status.SYMBOLS)],
  },
  {
    key: "telem",
    name: "Telemetry",
    Icon: Icon.Channel,
    symbols: [...Object.keys(Source.SYMBOLS)],
  },
  {
    key: "operator",
    name: "Operators",
    Icon: Icon.Add,
    symbols: Object.keys(Operator.SYMBOLS),
  },
  {
    key: "flow_control",
    name: "Flow Control",
    Icon: Icon.Select,
    symbols: [...Object.keys(Select.SYMBOLS), ...Object.keys(StableFor.SYMBOLS)],
  },
];
