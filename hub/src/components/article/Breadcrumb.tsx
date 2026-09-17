// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb as Base } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";

import { parseSegments } from "@/components/text/InlineCode";

export interface BreadcrumbProps {
  url: string;
  title?: string;
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
  http: "HTTP",
  and: "and",
  in: "in",
  sys: "System",
  admin: "Administrator",
};

const capitalize = (str: string): string =>
  breadcrumbOverrides[str] ?? caseconv.capitalize(str);

// The URL slug drops the casing of a code title, so a fully backticked title supplies
// the text of the last segment itself.
const codeTitle = (title?: string): string | null => {
  const segments = parseSegments(title);
  if (segments.length !== 1 || !segments[0].code) return null;
  return segments[0].text;
};

export const Breadcrumb = ({ url, title }: BreadcrumbProps) => {
  const code = codeTitle(title);
  const path = url.slice(1);
  const lastIndex = path.split("/").length - 1;
  return (
    <Base.Breadcrumb level="small" highlightVariant="last">
      {Base.mapURLSegments(path, ({ segment, href, index }) => (
        <Base.Segment href={`/${href}`} key={index}>
          {code != null && index === lastIndex ? (
            <code>{code}</code>
          ) : (
            segment.split("-").map(capitalize).join(" ")
          )}
        </Base.Segment>
      ))}
    </Base.Breadcrumb>
  );
};
