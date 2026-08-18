// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";
import { Zip, ZipPassThrough } from "fflate";

/** Reports whether the file at name is a zip archive by its extension. */
export const isZipFile = (name: string): boolean => name.toLowerCase().endsWith(".zip");

/**
 * A bundle source file: a path relative to the bundle root in forward-slash form and
 * lazily read bytes.
 */
export interface SourceFile {
  path: string;
  readBytes: () => Promise<Uint8Array<ArrayBuffer>>;
}

// The Core reads only files in a supported serialization extension from a bundle —
// members, the manifest, and a legacy LAYOUT.json are all .json — so anything else
// (repository junk, images) is dead upload weight.
const isBundleFile = ({ path }: SourceFile): boolean =>
  path.toLowerCase().endsWith(".json");

/**
 * Zips the source files into archive bytes ready to upload as a bundle. Files the Core
 * would ignore are dropped. The rest are read one at a time into a streaming zip, so
 * peak memory holds one file's bytes beside the archive. Entries are stored
 * uncompressed; the Core re-reads them anyway.
 */
export const zipFiles = async (
  files: SourceFile[],
): Promise<Uint8Array<ArrayBuffer>> => {
  const chunks: Uint8Array<ArrayBuffer>[] = [];
  await new Promise<void>((resolve, reject) => {
    const zip = new Zip((err, chunk, final) => {
      if (err != null) return reject(err);
      chunks.push(chunk);
      if (final) resolve();
    });
    void (async () => {
      try {
        for (const file of files.filter(isBundleFile)) {
          const entry = new ZipPassThrough(file.path);
          zip.add(entry);
          entry.push(await file.readBytes(), true);
        }
        zip.end();
      } catch (err) {
        reject(errors.fromUnknown(err));
      }
    })();
  });
  const archive = new Uint8Array(
    chunks.reduce((size, chunk) => size + chunk.length, 0),
  );
  let offset = 0;
  for (const chunk of chunks) {
    archive.set(chunk, offset);
    offset += chunk.length;
  }
  return archive;
};
