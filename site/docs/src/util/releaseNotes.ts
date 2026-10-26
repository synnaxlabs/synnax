// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The indexer runs under plain Node, so this module imports nothing from the site.
import fs from "fs";
import path from "path";

/** The directory that holds each version's notes. */
export const RELEASE_NOTES_DIR = path.join(import.meta.dirname, "../releases");

export interface Notes {
  /** The file name without its extension, which was the notes' route before. */
  name: string;
  /** The id of the notes on /releases. */
  anchor: string;
}

/** Returns the id that the notes of a version render under on /releases. */
export const releaseAnchor = (version: string): string => version.split(".").join("-");

const VERSION = /version="([^"]+)"/;

/**
 * Lists the notes in a directory.
 * @throws {Error} if a file does not pass a version to its release.
 */
export const listReleaseNotes = (dir: string = RELEASE_NOTES_DIR): Notes[] =>
  fs
    .readdirSync(dir)
    .filter((f) => f.endsWith(".mdx"))
    .map((f) => {
      const version = VERSION.exec(fs.readFileSync(path.join(dir, f), "utf8"))?.[1];
      if (version == null) throw new Error(`${f} passes no version to its release`);
      return { name: f.replace(/\.mdx$/, ""), anchor: releaseAnchor(version) };
    });
