// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";

import { Synch } from "@/synch";

export const useCommandSynchronizer = (
  onCommandUpdate: (command: task.Command) => void,
): void =>
  Synch.useStateChannel(task.COMMAND_CHANNEL_NAME, task.commandZ, onCommandUpdate);
