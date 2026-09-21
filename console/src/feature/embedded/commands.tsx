// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Status } from "@synnaxlabs/pluto";

import { showLogs } from "@/feature/embedded/supervisor";
import { Command } from "@/platform/command";

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

export const COMMANDS = [ShowLogsCommand];
