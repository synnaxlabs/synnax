// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@/feature/arc";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { type Selector } from "@/platform/selector";

export const VIS_SELECTABLES: Selector.Selectable[] = [
  ...LinePlot.SELECTABLES,
  ...Schematic.SELECTABLES,
  ...Log.SELECTABLES,
  ...Table.SELECTABLES,
];

export const SELECTABLES: Selector.Selectable[] = [
  ...VIS_SELECTABLES,
  ...Task.SELECTABLES,
  ...Arc.SELECTABLES,
];
