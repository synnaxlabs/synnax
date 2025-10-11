// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb as B } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";

export interface BreadcrumbProps {
  url: string;
}

export const Breadcrumb = ({ url }: BreadcrumbProps) => (
  <B.Breadcrumb level="small" highlightVariant="last">
    {B.mapURLSegments(url.slice(1), ({ segment, href, index }) => (
      <B.Segment href={`/${href}`} key={index}>
        {segment.split("-").map(caseconv.capitalize).join(" ")}
      </B.Segment>
    ))}
  </B.Breadcrumb>
);
