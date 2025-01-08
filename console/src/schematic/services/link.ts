// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";

import { type Link } from "@/link";
import { create } from "@/schematic/Schematic";
import { type State } from "@/schematic/slice";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  client,
  placer,
  addStatus,
}): Promise<boolean> => {
  if (resource !== "schematic") return false;
  try {
    const schematic = await client.workspaces.schematic.retrieve(resourceKey);
    const layoutCreator = create({
      ...(schematic.data as unknown as State),
      key: schematic.key,
      name: schematic.name,
      editable: false,
    });
    placer(layoutCreator);
  } catch (e) {
    Status.handleException(e, "Failed to open schematic from URL", addStatus);
  }
  return true;
};
