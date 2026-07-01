// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

import { retrieveAndPlaceLayout as retrieveAndPlaceTaskLayout } from "@/app/task/layouts";
import { type Range } from "@/platform/range";
import { create as createSchematic } from "@/platform/schematic/layout";

export const SNAPSHOT_SERVICES: Range.SnapshotServices = {
  schematic: {
    icon: <Icon.Schematic />,
    onClick: async ({ id: { key } }, { client, placeLayout }) => {
      if (client == null) throw new DisconnectedError();
      const s = await client.schematics.retrieve({ key });
      placeLayout(createSchematic({ key: s.key, name: s.name, editable: false }));
    },
    onDelete: async ({ id: { key } }, { client }) => {
      if (client == null) throw new DisconnectedError();
      await client.schematics.delete(key);
    },
  },
  task: {
    icon: <Icon.Task />,
    onClick: async ({ id: { key } }, { client, placeLayout }) =>
      await retrieveAndPlaceTaskLayout(client, key, placeLayout),
    onDelete: async ({ id: { key } }, { client }) => {
      if (client == null) throw new DisconnectedError();
      await client.tasks.delete(key);
    },
  },
};
