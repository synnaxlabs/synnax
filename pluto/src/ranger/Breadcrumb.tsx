// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { type CrudeTimeRange } from "@synnaxlabs/x";

import { Breadcrumb as Base } from "@/breadcrumb";
import { StageIcon } from "@/ranger/StageIcon";

export interface BreadcrumbProps extends Omit<Base.BreadcrumbProps, "children"> {
  timeRange?: CrudeTimeRange;
  name: string;
  showParent?: boolean;
  parent?: Pick<ranger.Payload, "name"> | null;
}

export const Breadcrumb = ({
  timeRange,
  name,
  parent,
  showParent = true,
  ...rest
}: BreadcrumbProps) => (
  <Base.Breadcrumb {...rest}>
    <Base.Segment weight={450} color={10}>
      {timeRange != null && <StageIcon timeRange={timeRange} />}
      {name}
    </Base.Segment>
    {parent != null && showParent && (
      <Base.Segment weight={400} color={9}>
        {parent.name}
      </Base.Segment>
    )}
  </Base.Breadcrumb>
);
