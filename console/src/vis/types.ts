// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Arc } from "@/arc";
import { type LinePlot } from "@/layered/service/lineplot";
import { type Log } from "@/layered/service/log";
import { type Schematic } from "@/layered/service/schematic";
import { type Table } from "@/layered/service/table";

export type LayoutType =
  | LinePlot.LayoutType
  | Log.LayoutType
  | Schematic.LayoutType
  | Table.LayoutType
  | Arc.EditorLayoutType;
