// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read } from "@/feature/labjack/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/feature/labjack/task/types";
import { Write } from "@/feature/labjack/task/Write";
import { type Task } from "@/platform/task";

export * from "@/feature/labjack/task/commands";
export * from "@/feature/labjack/task/integration";
export * from "@/feature/labjack/task/Read";
export * from "@/feature/labjack/task/SelectReadChannelTypeField";
export * from "@/feature/labjack/task/SelectWriteChannelType";
export * from "@/feature/labjack/task/types";
export * from "@/feature/labjack/task/Write";

export const FORMS: Task.Forms = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};
