// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";

import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { fromClientRange } from "@/range/services/ontology";
import { add, setActive } from "@/range/slice";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  client,
  dispatch,
  placer,
  addStatus,
  windowKey,
}): Promise<boolean> => {
  if (resource != "range") return false;
  try {
    const range = await client.ranges.retrieve(resourceKey);
    dispatch(setActive(range.key));
    add({ ranges: fromClientRange(range) });
    placer(
      LinePlot.create({
        name: `Plot for ${range.name}`,
        ranges: {
          x1: [range.key],
          x2: [],
        },
      }),
    );
    dispatch(Layout.setNavDrawerVisible({ windowKey, key: "range" }));
  } catch (e) {
    addStatus({
      variant: "error",
      key: id.id(),
      message: (e as Error).message,
    });
  }
  return true;
};
