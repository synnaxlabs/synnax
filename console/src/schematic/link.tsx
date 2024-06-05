// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Link } from "@/link";
import { create, State } from "@/schematic/slice";

export const linkHandler: Link.Handler = ({
  resource,
  resourceKey,
  client,
  placer,
}) => {
  if (resource != "schematic") return false;
  client.workspaces.schematic
    .retrieve(resourceKey)
    .then((schematic) => {
      if (schematic == null) return false;
      const layoutCreator = create({
        ...(schematic.data as unknown as State),
        key: schematic.key,
        name: schematic.name,
        // snapshot: schematic.snapshot,
      });
      placer(layoutCreator);
      return true;
    })
    .catch((error) => {
      console.error("Error: ", error);
      return false;
    });
  return false;
};
