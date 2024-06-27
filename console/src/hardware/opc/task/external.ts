// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReadTask } from "@/hardware/opc/task/ReadTask";
import { READ_TYPE } from "@/hardware/opc/task/types";
import { Layout } from "@/layout";

export * from "@/hardware/opc/task/ReadTask";
export * from "@/hardware/opc/task/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: ReadTask,
};
