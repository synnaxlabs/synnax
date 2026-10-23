// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

export interface Segment {
  text: string;
  code: boolean;
}

/** Splits text into plain and backtick-delimited inline code segments. */
export const parseSegments = (text?: string): Segment[] =>
  String(text ?? "")
    .split(/(`[^`]+`)/g)
    .filter((s) => s.length > 0)
    .map((s) =>
      s.startsWith("`") && s.endsWith("`")
        ? { text: s.slice(1, -1), code: true }
        : { text: s, code: false },
    );

/**
 * Rewrites backtick-delimited spans in an HTML string as inline code elements. Use for
 * text that must stay HTML, such as Algolia snippets that carry highlight markup.
 */
export const codifyHTML = (html?: string): string =>
  String(html ?? "").replace(/`([^`]+)`/g, "<code>$1</code>");

export interface SegmentsProps {
  segments: Segment[];
}

export const Segments = ({ segments }: SegmentsProps): ReactElement => (
  <>
    {segments.map(({ text, code }, i) => (code ? <code key={i}>{text}</code> : text))}
  </>
);

export interface InlineCodeProps {
  text?: string;
}

export const InlineCode = ({ text }: InlineCodeProps): ReactElement => (
  <Segments segments={parseSegments(text)} />
);
