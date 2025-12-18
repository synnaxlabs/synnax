// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { type Palette } from "@/palette";
import { create as createDashboard } from "@/perf/layout";
import * as Perf from "@/perf/slice";

const START_HARNESS_COMMAND: Palette.Command = {
  key: "start-perf-harness",
  name: "Start Console Profiling",
  icon: <Icon.Play />,
  visible: ({ state }) => state.perf?.status === "idle",
  onSelect: ({ store, placeLayout }) => {
    placeLayout(createDashboard());
    store.dispatch(Perf.start(undefined));
  },
};

const OPEN_DASHBOARD_COMMAND: Palette.Command = {
  key: "open-perf-dashboard",
  name: "Open Console Profiling Monitor",
  icon: <Icon.Units.Pressure />,
  onSelect: ({ placeLayout }) => {
    placeLayout(createDashboard());
  },
};

const STOP_HARNESS_COMMAND: Palette.Command = {
  key: "stop-perf-harness",
  name: "Stop Console Profiling",
  icon: <Icon.Pause />,
  visible: ({ state }) => state.perf?.status === "running",
  onSelect: ({ store }) => {
    store.dispatch(Perf.stop());
  },
};

const RESET_HARNESS_COMMAND: Palette.Command = {
  key: "reset-perf-harness",
  name: "Reset Console Profiling",
  icon: <Icon.Refresh />,
  visible: ({ state }) =>
    state.perf?.status === "completed" || state.perf?.status === "error",
  onSelect: ({ store }) => {
    store.dispatch(Perf.reset());
  },
};

export const COMMANDS: Palette.Command[] = [
  START_HARNESS_COMMAND,
  OPEN_DASHBOARD_COMMAND,
  STOP_HARNESS_COMMAND,
  RESET_HARNESS_COMMAND,
];
