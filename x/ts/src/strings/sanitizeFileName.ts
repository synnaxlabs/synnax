// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Matches the path separators, the control characters, and the characters Windows
// reserves.
const UNSAFE_FILE_NAME_CHARS = /[/\\<>:"|?*\p{Cc}]/gu;
const TRAILING_DOTS_AND_SPACES = /[. ]+$/;
// Matches the device names Windows refuses to open a file under, bare or carrying an
// extension.
const RESERVED_FILE_NAMES = /^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\.|$)/i;

/**
 * Turns a user-supplied name into a directory or file name that writes to disk on any
 * platform. Replaces every character a file name cannot hold with an underscore, drops
 * trailing dots and spaces, and prefixes an underscore to a Windows device name.
 *
 * @returns an empty string for a name that sanitizes to nothing, such as one holding
 * dots and spaces alone.
 */
export const sanitizeFileName = (name: string): string => {
  const sanitized = name
    .replace(UNSAFE_FILE_NAME_CHARS, "_")
    .replace(TRAILING_DOTS_AND_SPACES, "");
  return RESERVED_FILE_NAMES.test(sanitized) ? `_${sanitized}` : sanitized;
};
