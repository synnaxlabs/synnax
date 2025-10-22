// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb as Core } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";

export interface BreadcrumbProps {
  url: string;
}

// not exactly best coding practices but is a quick fix for the breadcrumb
// capitalization issues (SY-1468)
const breadcrumbOverrides: Record<string, string> = {
  cli: "CLI",
  systemd: "systemd",
  typescript: "TypeScript",
  ui: "UI",
  labjack: "LabJack",
  ni: "NI",
  opc: "OPC",
  ua: "UA",
  and: "and",
  in: "in",
  sys: "System",
  admin: "Administrator",
};

const capitalize = (str: string): string =>
  breadcrumbOverrides[str] ?? caseconv.capitalize(str);

export const Breadcrumb = ({ url }: BreadcrumbProps) => (
  <Core.Breadcrumb level="small" highlightVariant="last">
    {Core.mapURLSegments(url.slice(1), ({ segment, href, index }) => (
      <Core.Segment href={`/${href}`} key={index}>
        {segment.split("-").map(capitalize).join(" ")}
      </Core.Segment>
    ))}
  </Core.Breadcrumb>
);
