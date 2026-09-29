// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useConnectModal } from "@/feature/arinc429/device/device";
import { Task } from "@/feature/arinc429/task";
import { Bus } from "@/feature/bus";

export const ContextMenuItems = Bus.createContextMenuItems({
  integration: "arinc429",
  useConnectModal,
  useCreateRead: Task.useCreateRead,
  useCreateWrite: Task.useCreateWrite,
});
