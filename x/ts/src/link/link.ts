// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

const urlRegex = new RegExp(
  "^(https?:\\/\\/)?" + // http:// or https:// (optional)
    "((([a-zA-Z0-9][a-zA-Z0-9-]*\\.)+[a-zA-Z]{2,})|" + // domain name and extension
    "localhost|" + // localhost
    "(\\d{1,3}\\.){3}\\d{1,3})" + // or IP address
    "(\\:\\d+)?" + // port (optional)
    "(\\/[-a-zA-Z0-9@:%._\\+~#=]*)*" + // path (optional)
    "(\\?[;&a-zA-Z0-9%_.,~+=-]*)?" + // query string (optional)
    "(#[-a-zA-Z0-9_]*)?$", // fragment identifier (optional)
);

export const is = (string: string): boolean => urlRegex.test(string);

// End punctuation and an unmatched parenthesis belong to the sentence around a link.
const groupPattern = /\([^\s"'<>()]*\)/.source;
const charPattern = `(?:[^\\s"'<>()]|${groupPattern})`;
const tailPattern = `${charPattern}*(?:[^\\s"'<>().,;:!?\\]}]|${groupPattern})`;
const urlPattern = `(?:(?:https?|synnax):\\/\\/|\\bwww\\.)${tailPattern}`;
const octetPattern = /(?:25[0-5]|2[0-4]\d|1?\d?\d)/.source;
// An address stands alone, so v1.2.3.4, tcp://1.2.3.4, and 1.2.3.0/24 hold none.
const addressPattern =
  `(^|[^\\w./])((?:${octetPattern}\\.){3}${octetPattern}(?!\\/\\d{1,2}(?![\\w/]))` +
  `(?::\\d{1,5})?(?:\\/${tailPattern})?)(?!\\w|\\.\\d)`;
// A path is absolute and starts a word, so and/or holds none.
const pathStartPattern = /(?:~?\/(?=[\w.~-])|[A-Za-z]:[\\/])/.source;
const pathPattern = `(^|[\\s("'=[])(${pathStartPattern}${tailPattern})`;
const embeddedRegex = new RegExp(
  `(${urlPattern})|${addressPattern}|${pathPattern}`,
  "g",
);

export const linkZ = z.object({ kind: z.enum(["url", "path"]), target: z.string() });

/** A URL or a file path to open. */
export interface Link extends z.infer<typeof linkZ> {}

/** How a link opens. */
export type Kind = Link["kind"];

/** A link in a text, from start up to end. */
export interface Span extends Link {
  start: number;
  end: number;
  /** True for a Markdown link, which shows its label and not its target. */
  labeled: boolean;
}

const find = (text: string): Omit<Span, "labeled">[] =>
  Array.from(text.matchAll(embeddedRegex), (match) => {
    const { 1: url, 2: addressLead, 3: address, 4: pathLead, 5: path, index } = match;
    const start = index + (addressLead ?? pathLead ?? "").length;
    const raw = url ?? address ?? path;
    let scheme = "";
    if (address != null) scheme = "http://";
    else if (raw.startsWith("www.")) scheme = "https://";
    const kind: Kind = path == null ? "url" : "path";
    return { start, end: start + raw.length, kind, target: scheme + raw };
  });

const labelRegex = /\[([^[\]]+)\]\($/;

/** A text as it shows, and the links in it. */
export interface Parsed {
  text: string;
  spans: Span[];
}

/** @returns the text with each Markdown link as its label, and the links in it. */
export const parse = (raw: string): Parsed => {
  const spans: Span[] = [];
  let text = "";
  let last = 0;
  for (const { start, end, kind, target } of find(raw)) {
    let [from, to, shown] = [start, end, raw.slice(start, end)];
    const label = raw[end] === ")" ? labelRegex.exec(raw.slice(last, start)) : null;
    if (label != null) [from, to, shown] = [last + label.index, end + 1, label[1]];
    text += raw.slice(last, from) + shown;
    const labeled = label != null;
    spans.push({
      start: text.length - shown.length,
      end: text.length,
      kind,
      target,
      labeled,
    });
    last = to;
  }
  return { text: text + raw.slice(last), spans };
};
