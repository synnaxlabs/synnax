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
import { useCallback } from "react";

import { DEFAULT_TRIGGERED } from "@/feature/lineplot/CustomRangeInput";
import { FLAGS } from "@/flags";
import { Command } from "@/platform/command";
import { LinePlot as PlatformLinePlot } from "@/platform/lineplot";
import { Range } from "@/platform/range";

const CreateCommand = Command.create({
  key: "create_line_plot",
  name: "Create line plot",
  icon: <LinePlot.CreateIcon />,
  useOnSelect: PlatformLinePlot.useCreate,
  useVisible: () => Access.useCreateGranted(lineplot.TYPE_ONTOLOGY_ID),
});

const useCreateOscilloscope = (): (() => void) => {
  const create = PlatformLinePlot.useCreate({ toolbarTab: "measure" });
  return useCallback(
    () =>
      create({
        name: "Oscilloscope",
        ranges: { x1: [Range.CUSTOM_KEY], x2: [], custom: DEFAULT_TRIGGERED },
      }),
    [create],
  );
};

const CreateOscilloscopeCommand = Command.create({
  key: "create_oscilloscope",
  name: "Create oscilloscope",
  icon: <LinePlot.CreateIcon />,
  useOnSelect: useCreateOscilloscope,
  useVisible: () =>
    FLAGS.lineplotWindows && Access.useCreateGranted(lineplot.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, CreateOscilloscopeCommand];
