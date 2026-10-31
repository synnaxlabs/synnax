// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb as Base } from "@synnaxlabs/lyra/breadcrumb";
import { caseconv } from "@synnaxlabs/x";

import { Text } from "@/components/text";
import { labelOf } from "@/pages/_nav";

export interface BreadcrumbProps {
  url: string;
  title?: string;
}

// Acronyms and proper nouns in a URL slug, for routes that the nav does not name.
const WORDS: Record<string, string> = {
  arc: "Arc",
  cli: "CLI",
  ethercat: "EtherCAT",
  http: "HTTP",
  labjack: "LabJack",
  ni: "NI",
  opc: "OPC",
  pagerduty: "PagerDuty",
  python: "Python",
  synnax: "Synnax",
  typescript: "TypeScript",
  ua: "UA",
  ui: "UI",
};

const fromSlug = (segment: string): string =>
  segment
    .split("-")
    .map((word, i) => WORDS[word] ?? (i === 0 ? caseconv.capitalize(word) : word))
    .join(" ");

// The URL slug drops the casing of a code title, so a fully backticked title supplies
// the text of the last segment itself.
const codeTitle = (title?: string): string | null => {
  const segments = Text.parseSegments(title);
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
            (labelOf(`/${href}`) ?? fromSlug(segment))
          )}
        </Base.Segment>
      ))}
    </Base.Breadcrumb>
  );
};
