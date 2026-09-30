// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The browser imports this file, so it must not import anything that reaches the
// database.

export const MAX_NAME_LENGTH = 64;

/** VERSION_PATTERN matches a major.minor Core version, like 0.62. */
export const VERSION_PATTERN = /^\d+\.\d+$/;

/** KEY_FILE_EXTENSION is what the Console's file picker filters on. */
export const KEY_FILE_EXTENSION = "license";

/** filename is the name a license key downloads as, derived from its license label. */
export const filename = (label: string): string => {
  const stem = label.replace(/[^a-z0-9]+/gi, "-").replace(/^-|-$/g, "") || "synnax";
  return `${stem}.${KEY_FILE_EXTENSION}`;
};
