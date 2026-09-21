// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Status } from "@synnaxlabs/pluto";

import { useDiagnosticsModal } from "@/feature/embedded/Diagnostics";
import { showData, showLogs } from "@/feature/embedded/supervisor";
import { useReset } from "@/feature/embedded/useReset";
import { useRestart } from "@/feature/embedded/useRestart";
import { Command } from "@/platform/command";

export const OpenDiagnosticsCommand = Command.create({
  key: "open-diagnostics",
  name: "Open diagnostics",
  icon: <Icon.Hardware />,
  useVisible: () => true,
  useOnSelect: useDiagnosticsModal,
});

export const RestartCommand = Command.create({
  key: "restart-synnax",
  name: "Restart Synnax",
  icon: <Icon.Refresh />,
  useVisible: () => true,
  useOnSelect: useRestart,
});

export const ShowLogsCommand = Command.create({
  key: "show-logs",
  name: "Show logs",
  icon: <Icon.Log />,
  useVisible: () => true,
  useOnSelect: () => {
    const handleError = Status.useErrorHandler();
    return () => handleError(showLogs, "Failed to show the logs");
  },
});

export const ShowDataCommand = Command.create({
  key: "show-data-folder",
  name: "Show data folder",
  icon: <Icon.Explore />,
  useVisible: () => true,
  useOnSelect: () => {
    const handleError = Status.useErrorHandler();
    return () => handleError(showData, "Failed to show the data folder");
  },
});

export const EraseDataCommand = Command.create({
  key: "erase-all-data",
  name: "Erase all data",
  icon: <Icon.Delete />,
  useVisible: () => true,
  useOnSelect: useReset,
});

export const COMMANDS = [
  OpenDiagnosticsCommand,
  RestartCommand,
  ShowLogsCommand,
  ShowDataCommand,
  EraseDataCommand,
];
