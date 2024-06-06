// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Link } from "@/link";
import { setActive } from "@/range/slice";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  client,
  dispatch,
  addStatus,
}): Promise<boolean> => {
  if (resource != "range") return false;
  try {
    const range = await client.ranges.retrieve(resourceKey);
    dispatch(setActive(range.key));
  } catch (e) {
    addStatus({
      variant: "error",
      key: `openUrlError-${resource + "/" + resourceKey}`,
      message: (e as Error).message,
    });
  }
  return true;
};
