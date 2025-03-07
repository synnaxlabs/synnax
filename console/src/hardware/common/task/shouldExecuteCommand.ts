// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  PAUSED_STATUS,
  RUNNING_STATUS,
  START_COMMAND,
  type Status,
  STOP_COMMAND,
} from "@/hardware/common/task/types";

export const shouldExecuteCommand = (status: Status, command: string): boolean =>
  (status === RUNNING_STATUS && command === STOP_COMMAND) ||
  (status === PAUSED_STATUS && command === START_COMMAND);
