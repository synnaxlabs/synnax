// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { type Link } from "@/link";
import { overviewLayout } from "@/range/external";
import { add, setActive } from "@/range/slice";
import { fromClientRange } from "@/range/Toolbar";

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
    dispatch(add({ ranges: fromClientRange(range) }));
    placer({ ...overviewLayout, key: resourceKey });
    dispatch(Layout.setNavDrawerVisible({ windowKey, key: "range" }));
  } catch (e) {
    Status.handleException(e, "Failed to open range from URL", addStatus);
  }
  return true;
};
