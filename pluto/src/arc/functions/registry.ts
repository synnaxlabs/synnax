// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Constant } from "@/arc/functions/constant";
import { Operator } from "@/arc/functions/operator";
import { Select } from "@/arc/functions/select";
import { Sink } from "@/arc/functions/sink";
import { Source } from "@/arc/functions/source";
import { StableFor } from "@/arc/functions/stable";
import { Status } from "@/arc/functions/status";
import { type Spec } from "@/arc/functions/types/spec";
import { Icon } from "@/icon";

export const REGISTRY: Record<string, Spec<any>> = {
  ...Source.SYMBOLS,
  ...Sink.SYMBOLS,
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
    symbols: [...Object.keys(Source.SYMBOLS), ...Object.keys(Sink.SYMBOLS)],
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
