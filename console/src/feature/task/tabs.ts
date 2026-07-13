// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type Synnax, type task } from "@synnaxlabs/client";

import { type Panel } from "@/platform/panel";

export const retrieveAndOpenTab = async (
  client: Synnax | null,
  key: task.Key,
  openTab: Panel.OpenTab,
) => {
  if (client == null) throw new DisconnectedError();
  const t = await client.tasks.retrieve({ key });
  openTab({ variant: "view", type: t.type, args: { taskKey: key } });
};
