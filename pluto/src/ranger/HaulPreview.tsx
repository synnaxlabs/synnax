// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { color, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface HaulPreviewProps {
  range: ranger.Payload;
  /** How many items the drag carries. */
  count?: number;
}

/**
 * The drag preview of a range: its color, name, start, and length. A planned range has
 * no start, so it shows "Planned".
 */
export const HaulPreview = ({ range, count }: HaulPreviewProps): ReactElement => {
  const tr = new TimeRange(range.timeRange).makeValid();
  const facts: string[] = [];
  if (tr.start.equals(TimeStamp.MAX)) facts.push("Planned");
  else {
    facts.push(tr.start.toString("dateTime", "local"));
    if (!tr.end.equals(TimeStamp.MAX)) facts.push(tr.span.toString("semantic"));
  }
  return (
    <Haul.Preview
      icon={<Icon.Range />}
      swatch={color.cssString(range.color)}
      name={range.name}
      facts={facts}
      count={count}
    />
  );
};
