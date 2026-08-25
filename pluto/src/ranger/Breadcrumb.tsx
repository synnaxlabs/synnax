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
import { Text } from "@/text";

export interface BreadcrumbProps extends Omit<Base.BreadcrumbProps, "children"> {
  timeRange?: CrudeTimeRange;
  name: string;
  showParent?: boolean;
  parent?: Pick<ranger.Payload, "name"> | null;
  /** DOM id for the name, so {@link Text.edit} can target it. */
  nameID?: string;
  /** Makes the name editable in place. Absent, the name is plain text. */
  onRename?: (name: string) => void;
}

export const Breadcrumb = ({
  timeRange,
  name,
  parent,
  showParent = true,
  nameID,
  onRename,
  ...rest
}: BreadcrumbProps) => (
  <Base.Breadcrumb {...rest}>
    <Base.Segment weight={450} color={10}>
      {timeRange != null && <StageIcon timeRange={timeRange} />}
      <Text.MaybeEditable
        id={nameID}
        value={name}
        defaultEl="span"
        onChange={onRename}
        allowDoubleClick={false}
        weight={450}
        color={10}
      />
    </Base.Segment>
    {parent != null && showParent && (
      <Base.Segment weight={400} color={9}>
        {parent.name}
      </Base.Segment>
    )}
  </Base.Breadcrumb>
);
