// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Access, LinePlot } from "@synnaxlabs/pluto";

import { LinePlot as PlatformLinePlot } from "@/platform/lineplot";
import { Palette } from "@/platform/palette";

const CreateCommand = Palette.createCommand({
  key: "create_line_plot",
  name: "Create a line plot",
  icon: <LinePlot.CreateIcon />,
  useOnSelect: PlatformLinePlot.useCreate,
  useVisible: () => Access.useCreateGranted(lineplot.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand];
