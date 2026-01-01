// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Link } from "@/link";
import { Range } from "@/range";

export const handleLink: Link.Handler = async ({
  client,
  dispatch,
  key,
  placeLayout,
}) => {
  const range = await client.ranges.retrieve(key);
  dispatch(Range.add({ ranges: Range.fromClientRange(range) }));
  dispatch(Range.setActive(range.key));
  placeLayout({ ...Range.OVERVIEW_LAYOUT, key, name: range.name });
};
