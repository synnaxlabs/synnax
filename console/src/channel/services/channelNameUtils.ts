// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Regex pattern for allowed characters in channel names: alphanumeric, underscores, and dashes
 */
export const CHANNEL_NAME_ALLOWED_PATTERN = /^[a-zA-Z0-9_-]$/;

/**
 * Extracts the base name from a channel name by removing _cmd or _state suffixes
 */
export const extractBaseName = (name: string): string => {
  if (name.endsWith("_cmd")) return name.slice(0, -4);
  if (name.endsWith("_state")) return name.slice(0, -6);
  return name;
};

/**
 * Cleans up a channel name by:
 * - Squashing multiple underscores to single underscore
 * - Squashing multiple dashes to single dash  
 * - Removing trailing underscores and dashes
 * - Converting to lowercase
 */
export const cleanChannelName = (name: string): string =>
  name
    .replace(/_+/g, '_')    // Replace multiple underscores with single underscore
    .replace(/-+/g, '-')    // Replace multiple dashes with single dash
    .replace(/[_-]+$/, '')  // Remove trailing underscores and dashes
    .toLowerCase();         // Convert to lowercase

/**
 * Checks if a single character is allowed in channel names
 */
export const isAllowedChannelNameCharacter = (char: string): boolean =>
  CHANNEL_NAME_ALLOWED_PATTERN.test(char);