// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { zip } from "@synnaxlabs/x";

import { type FS } from "@/platform/fs";

/** Reports whether the file at name is a zip archive by its extension. */
export const isZipFile = (name: string): boolean => name.toLowerCase().endsWith(".zip");

// The Core reads only files in a supported serialization extension from a bundle —
// members, the manifest, and a legacy LAYOUT.json are all .json — so anything else
// (repository junk, images) is dead upload weight.
const isBundleFile = ({ path }: FS.SourceFile): boolean =>
  path.toLowerCase().endsWith(".json");

/**
 * Zips the source files into a byte stream ready to upload as a bundle. Files the Core
 * would ignore are dropped. Entries are stored uncompressed; the Core re-reads them
 * anyway.
 */
export const zipFiles = (files: FS.SourceFile[]): ReadableStream<Uint8Array> =>
  zip.create(files.filter(isBundleFile));
