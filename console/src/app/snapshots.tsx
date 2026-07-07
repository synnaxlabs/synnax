// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, schematic } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

import { type Range } from "@/platform/range";

// TODO(SY-4370): snapshot tabs opened here were previously marked non-editable;
// read-only tab state needs a panel equivalent.
export const SNAPSHOT_SERVICES: Range.SnapshotServices = {
  schematic: {
    icon: <Icon.Schematic />,
    onClick: async ({ id: { key } }, { openTab }) =>
      openTab({ variant: "resource", resource: schematic.ontologyID(key) }),
    onDelete: async ({ id: { key } }, { client }) => {
      if (client == null) throw new DisconnectedError();
      await client.schematics.delete(key);
    },
  },
  task: {
    icon: <Icon.Task />,
    onClick: async ({ id: { key } }, { client, openTab }) => {
      if (client == null) throw new DisconnectedError();
      const t = await client.tasks.retrieve({ key });
      openTab({ variant: "view", type: t.type, args: { taskKey: t.key } });
    },
    onDelete: async ({ id: { key } }, { client }) => {
      if (client == null) throw new DisconnectedError();
      await client.tasks.delete(key);
    },
  },
};
