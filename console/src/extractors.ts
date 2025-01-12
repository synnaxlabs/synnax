// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Schematic } from "@/schematic";
import { Table } from "@/table";

export const EXTRACTORS: Record<string, Export.Extractor> = {
  [LinePlot.LAYOUT_TYPE]: LinePlot.extract,
  [Log.LAYOUT_TYPE]: Log.extract,
  [Schematic.LAYOUT_TYPE]: Schematic.extract,
  [Table.LAYOUT_TYPE]: Table.extract,
};
