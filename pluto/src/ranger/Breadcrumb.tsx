// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";

import { Breadcrumb as Core } from "@/breadcrumb";

export interface BreadcrumbProps extends Omit<Core.BreadcrumbProps, "children"> {
  name: string;
  showParent?: boolean;
  parent?: Pick<ranger.Payload, "name"> | null;
}

export const Breadcrumb = ({
  name,
  parent,
  showParent = true,
  ...rest
}: BreadcrumbProps) => (
  <Core.Breadcrumb {...rest}>
    <Core.Segment weight={450} color={10}>
      {name}
    </Core.Segment>
    {parent != null && showParent && (
      <Core.Segment weight={400} color={9}>
        {parent.name}
      </Core.Segment>
    )}
  </Core.Breadcrumb>
);
