// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { zipSync } from "fflate";

/** Reports whether the file at name is a zip archive by its extension. */
export const isZipFile = (name: string): boolean => name.toLowerCase().endsWith(".zip");

export interface BundleFile {
  /** The file's path relative to the bundle root, forward-slash form. */
  path: string;
  bytes: Uint8Array<ArrayBuffer>;
}

/**
 * Zips the files into an archive keyed by path, ready to upload as a bundle. Entries
 * are stored uncompressed; bundle files are small and the Core re-reads them anyway.
 */
export const zipFiles = (files: BundleFile[]): Uint8Array<ArrayBuffer> =>
  zipSync(Object.fromEntries(files.map(({ path, bytes }) => [path, bytes])), {
    level: 0,
  });
