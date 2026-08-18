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

import { type FS } from "@/platform/fs";
import { Runtime } from "@/platform/runtime";

/** Reports whether the file at name is a zip archive by its extension. */
export const isZipFile = (name: string): boolean => name.toLowerCase().endsWith(".zip");

// The Core reads only files in a supported serialization extension from a bundle —
// members, the manifest, and a legacy LAYOUT.json are all .json — so anything else
// (repository junk, images) is dead upload weight.
const isBundleFile = ({ path }: FS.SourceFile): boolean =>
  path.toLowerCase().endsWith(".json");

/**
 * Zips the source files into a byte stream ready to upload as a bundle. Files the Core
 * would ignore are dropped. The rest are read one at a time into a streaming zip, so
 * peak memory holds one file's bytes beside the chunks the consumer has not drained.
 * Entries are stored uncompressed; the Core re-reads them anyway.
 */
export const zipFiles = (files: FS.SourceFile[]): ReadableStream<Uint8Array> =>
  new ReadableStream({
    start(controller) {
      const zip = new Zip((err, chunk, final) => {
        if (err != null) return controller.error(err);
        controller.enqueue(chunk);
        if (final) controller.close();
      });
      void (async () => {
        try {
          for (const file of files.filter(isBundleFile)) {
            const entry = new ZipPassThrough(file.path);
            zip.add(entry);
            entry.push(await Runtime.toBytes(await file.read()), true);
          }
          zip.end();
        } catch (err) {
          controller.error(errors.fromUnknown(err));
        }
      })();
    },
  });
