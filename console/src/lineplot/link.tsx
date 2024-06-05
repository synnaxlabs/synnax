// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { create, State } from "@/lineplot/slice";
import { Link } from "@/link";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  client,
  placer,
}): Promise<boolean> => {
  if (resource != "lineplot") return false;
  try {
    const linePlot = await client.workspaces.linePlot.retrieve(resourceKey);
    if (linePlot == null) return false;
    const layoutCreator = create({
      ...(linePlot.data as unknown as State),
      key: linePlot.key,
      name: linePlot.name,
    });
    placer(layoutCreator);
    return true;
  } catch (error) {
    console.error("Error: ", error);
    return false;
  }
};
