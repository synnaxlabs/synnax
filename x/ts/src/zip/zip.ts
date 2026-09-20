// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Zip, ZipPassThrough } from "fflate";

import { errors } from "@/errors";

/** A source file: a path relative to the archive root in forward-slash form, read
 * lazily. */
export interface Source {
  path: string;
  /** Reads the file's contents. */
  read: () => Promise<Uint8Array<ArrayBuffer> | Blob>;
}

const toBytes = async (
  data: Uint8Array<ArrayBuffer> | Blob,
): Promise<Uint8Array<ArrayBuffer>> =>
  data instanceof Uint8Array ? data : new Uint8Array(await data.arrayBuffer());

/**
 * Zips the source files into a byte stream. Files are read one at a time into a
 * streaming zip, so peak memory holds one file's bytes beside the chunks the consumer
 * has not drained. Entries are stored uncompressed. A source whose read fails errors
 * the stream.
 */
export const create = (files: Source[]): ReadableStream<Uint8Array> =>
  new ReadableStream({
    start(controller) {
      const zip = new Zip((err, chunk, final) => {
        if (err != null) return controller.error(err);
        controller.enqueue(chunk);
        if (final) controller.close();
      });
      void (async () => {
        try {
          for (const file of files) {
            const entry = new ZipPassThrough(file.path);
            zip.add(entry);
            entry.push(await toBytes(await file.read()), true);
          }
          zip.end();
        } catch (err) {
          controller.error(errors.fromUnknown(err));
        }
      })();
    },
  });
