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

// Punctuation at the end belongs to the sentence around a link.
const tailPattern = /[^\s"'<>]*[^\s"'<>.,;:!?)\]}]/.source;
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
}

/**
 * @returns each link in the text, in order. An address or a www name gets a scheme. A
 * closing parenthesis stays in a link only when the link opens it.
 */
export const find = (text: string): Span[] =>
  Array.from(text.matchAll(embeddedRegex), (match): Span => {
    const { 1: url, 2: addressLead, 3: address, 4: pathLead, 5: path, index } = match;
    const start = index + (addressLead ?? pathLead ?? "").length;
    const body = url ?? address ?? path;
    let end = start + body.length;
    let open = body.split("(").length - body.split(")").length;
    for (; open > 0 && text[end] === ")"; open--) end++;
    const raw = text.slice(start, end);
    let scheme = "";
    if (address != null) scheme = "http://";
    else if (raw.startsWith("www.")) scheme = "https://";
    return { start, end, kind: path == null ? "url" : "path", target: scheme + raw };
  });
