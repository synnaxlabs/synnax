// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Matches the path separators, the characters Windows reserves, and the control
// characters Windows forbids.
// eslint-disable-next-line no-control-regex
const UNSAFE_FILE_NAME_CHARS = /[/\\<>:"|?*\x00-\x1f]/gu;
const TRAILING_DOTS_AND_SPACES = /[. ]+$/;
// Matches the device names Windows refuses to open a file under, bare or carrying an
// extension.
const RESERVED_FILE_NAMES = /^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\.|$)/i;
// The longest single path element ext4, APFS, and NTFS accept. It counts bytes, which
// bounds NTFS's UTF-16 limit too: a code point never takes more UTF-16 code units than
// it takes UTF-8 bytes.
const MAX_FILE_NAME_LENGTH = 255;
// Names a file whose name sanitizes to nothing.
const PLACEHOLDER_FILE_NAME = "_";

const encoder = new TextEncoder();

const utf8Length = (value: string): number => encoder.encode(value).length;

/**
 * Shortens name to maxBytes bytes, cutting on a code point boundary, and drops the
 * trailing dots and spaces Windows drops. It trims after cutting because the cut can
 * expose a dot or a space the original name buried.
 */
const fit = (name: string, maxBytes: number): string => {
  if (maxBytes <= 0) return "";
  let fitted = "";
  let bytes = 0;
  for (const codePoint of name) {
    const size = utf8Length(codePoint);
    if (bytes + size > maxBytes) break;
    bytes += size;
    fitted += codePoint;
  }
  return fitted.replace(TRAILING_DOTS_AND_SPACES, "");
};

/**
 * Turns a user-supplied name into a directory or file name that writes to disk on any
 * platform, carrying extension. Replaces every character a file name cannot hold with
 * an underscore, drops trailing dots and spaces, prefixes an underscore to a Windows
 * device name or a name starting with a dot, and shortens the name until it and
 * extension together fit the longest path element a filesystem takes. The result is a
 * single path element, but it is not unique: two names can sanitize to one, and
 * shortening makes that more likely.
 * @param extension - The extension the result carries, leading dot included. Defaults
 * to none.
 * @returns a single underscore for a name that sanitizes to nothing, such as one
 * holding dots and spaces alone.
 * @throws {Error} if extension fills a file name by itself, which no name can rescue.
 */
export const sanitize = (name: string, extension: string = ""): string => {
  const budget = MAX_FILE_NAME_LENGTH - utf8Length(extension);
  if (budget <= 0)
    throw new Error(`extension "${extension}" leaves no room for a file name`);
  let sanitized = fit(name.replace(UNSAFE_FILE_NAME_CHARS, "_"), budget);
  // A leading dot hides the file on Unix-like systems, so it gets a prefix too. Hold a
  // byte back for the prefix so the whole name still fits.
  if (RESERVED_FILE_NAMES.test(sanitized) || sanitized.startsWith("."))
    sanitized = `_${fit(sanitized, budget - 1)}`;
  if (sanitized === "") sanitized = PLACEHOLDER_FILE_NAME;
  return sanitized + extension;
};
