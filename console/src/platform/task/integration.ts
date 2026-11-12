// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Icon } from "@synnaxlabs/lyra/icon";

import { type Selector } from "@/platform/selector";
import { type PlotChannels } from "@/platform/task/plot";

/** The tasks of an integration, as the task feature names and lists them. */
export interface Integration {
  /** The first word of each of its task types, such as "opc". */
  prefix: string;
  /** The display name, such as "OPC UA". */
  name: string;
  icon: Icon.ReactElement;
  /** The task types a user can create. */
  selectables: Selector.Selectable[];
  /** The read task types that a plot can show, by type. */
  plotChannels?: Record<string, PlotChannels>;
}
