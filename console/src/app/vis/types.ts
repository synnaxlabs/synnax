// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Arc } from "@/feature/arc";
import { type LinePlot } from "@/feature/lineplot";
import { type Log } from "@/feature/log";
import { type Schematic } from "@/feature/schematic";
import { type Table } from "@/feature/table";

export type LayoutType =
  | LinePlot.LayoutType
  | Log.LayoutType
  | Schematic.LayoutType
  | Table.LayoutType
  | Arc.EditorLayoutType;
