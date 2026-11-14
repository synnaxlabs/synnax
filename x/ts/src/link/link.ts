// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
const embeddedRegex = /https?:\/\/[^\s"'<>]*[^\s"'<>.,;:!?)\]}]/g;

/** A link in a text, from start up to end. */
export interface Span {
  start: number;
  end: number;
  url: string;
}

/**
 * @returns each link in the text, in order. A closing parenthesis stays in a link only
 * when the link opens it.
 */
export const find = (text: string): Span[] =>
  Array.from(text.matchAll(embeddedRegex), ({ 0: match, index }) => {
    let end = index + match.length;
    let open = match.split("(").length - match.split(")").length;
    for (; open > 0 && text[end] === ")"; open--) end++;
    return { start: index, end, url: text.slice(index, end) };
  });
