// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const NAMED: Record<string, string> = {
  "\n": "\\n",
  "\r": "\\r",
  "\t": "\\t",
  "\0": "\\0",
  "\\": "\\\\",
};

const UNNAMED: Record<string, string> = Object.fromEntries(
  Object.entries(NAMED).map(([raw, escaped]) => [escaped.slice(1), raw]),
);

// Printable ASCII and characters beyond one byte read as themselves.
const isPlain = (c: string): boolean => (c >= " " && c <= "~") || c > "\xff";

/** @returns text with control and other single-byte characters written as escapes. */
export const formatEscaped = (text: string): string =>
  Array.from(text, (c) => {
    if (c in NAMED) return NAMED[c];
    if (isPlain(c)) return c;
    return `\\x${c.charCodeAt(0).toString(16).padStart(2, "0")}`;
  }).join("");

/** @returns text with the escapes formatEscaped writes turned back into characters. */
export const parseEscaped = (text: string): string =>
  text.replace(/\\(x[0-9a-fA-F]{2}|[nrt0\\])/g, (_, code: string) =>
    code.startsWith("x")
      ? String.fromCharCode(parseInt(code.slice(1), 16))
      : UNNAMED[code],
  );
