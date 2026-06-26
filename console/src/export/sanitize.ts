// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const UNSAFE_FILE_NAME_CHARS = /[/\\<>:"|?*]/g;

/**
 * Replaces characters that are unsafe in filenames across platforms (path
 * separators and Windows-reserved characters) with underscores. Used by
 * export flows that turn user-supplied names into directory and file names
 * on disk.
 */
export const sanitizeFileName = (name: string): string =>
  name.replace(UNSAFE_FILE_NAME_CHARS, "_");
