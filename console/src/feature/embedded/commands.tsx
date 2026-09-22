// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { showData, showLogs } from "@/feature/embedded/supervisor";
import { useDiagnosticsModal } from "@/feature/embedded/useDiagnosticsModal";
import { useReset } from "@/feature/embedded/useReset";
import { useRestart } from "@/feature/embedded/useRestart";
import { useReveal } from "@/feature/embedded/useReveal";
import { Command } from "@/platform/command";

export const OpenDiagnosticsCommand = Command.create({
  key: "open-diagnostics",
  name: "Open diagnostics",
  icon: <Icon.Hardware />,
  useOnSelect: useDiagnosticsModal,
});

export const RestartCommand = Command.create({
  key: "restart-synnax",
  name: "Restart Synnax",
  icon: <Icon.Refresh />,
  useOnSelect: useRestart,
});

export const ShowLogsCommand = Command.create({
  key: "show-logs",
  name: "Show logs",
  icon: <Icon.Log />,
  useOnSelect: () => useReveal(showLogs, "Failed to show the logs"),
});

export const ShowDataCommand = Command.create({
  key: "show-data-folder",
  name: "Show data folder",
  icon: <Icon.Explore />,
  useOnSelect: () => useReveal(showData, "Failed to show the data folder"),
});

export const EraseDataCommand = Command.create({
  key: "erase-all-data",
  name: "Erase all data",
  icon: <Icon.Delete />,
  useOnSelect: useReset,
});

export const COMMANDS = [
  OpenDiagnosticsCommand,
  RestartCommand,
  ShowLogsCommand,
  ShowDataCommand,
  EraseDataCommand,
];
