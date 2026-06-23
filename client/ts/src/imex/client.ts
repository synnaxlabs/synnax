// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FileClient, type UploadBody } from "@synnaxlabs/freighter";

import { ontology } from "@/ontology";

const IMPORT_ENDPOINT = "/imex/import";
const EXPORT_ENDPOINT = "/imex/export";

const DEFAULT_CONTENT_TYPE = "application/json";

/**
 * Imports and exports resources to and from a Synnax cluster as serialized bytes — the
 * same on-disk representation a user reads from or writes to a file. The bytes are
 * opaque to the client: nothing is parsed or transformed on the way through, so the wire
 * format is exactly the file's contents.
 *
 * Each call moves exactly one resource, and large payloads are streamed end-to-end. The
 * client is environment-agnostic — the byte streams are standard Web Streams that work in
 * browsers, Node 18+, and Tauri webviews. Callers bridge the filesystem with their own
 * runtime's primitives:
 *
 * - Node: `import(Readable.toWeb(createReadStream(path)))` and
 *   `(await export(id)).pipeTo(Writable.toWeb(createWriteStream(path)))`.
 * - Browser: `import(fileInput.files[0])` and pipe the export into a
 *   showSaveFilePicker writable (or buffer it with `new Response(stream).blob()`).
 * - Tauri/Console: read the picked file into a Blob to import, and hand the export
 *   stream to the Console's downloadStream helper.
 */
export class Client {
  private readonly file: FileClient;

  constructor(file: FileClient) {
    this.file = file;
  }

  /**
   * Imports a resource from the given serialized source and returns its new ontology id.
   *
   * @param source - the serialized resource (e.g. a file's contents). A ReadableStream or
   * Blob is streamed to the Core without being buffered into memory; an ArrayBufferView
   * or string is sent as-is.
   * @param contentType - the wire format of source. Defaults to "application/json".
   * @returns the new resource's ontology id as stamped by the Core.
   */
  async import(
    source: UploadBody,
    contentType: string = DEFAULT_CONTENT_TYPE,
  ): Promise<ontology.ID> {
    return await this.file.upload(IMPORT_ENDPOINT, source, contentType, ontology.idZ);
  }

  /**
   * Exports the resource identified by id as a byte stream. The caller pipes the stream
   * wherever it likes — a file on disk, a browser download, or an in-memory buffer via
   * `new Response(stream).json()` — without the client ever buffering the whole payload.
   *
   * @param id - the ontology id of the resource to export.
   * @returns the serialized resource as a stream of bytes.
   */
  async export(id: ontology.ID): Promise<ReadableStream<Uint8Array>> {
    return await this.file.download(
      EXPORT_ENDPOINT,
      id,
      ontology.idZ,
      DEFAULT_CONTENT_TYPE,
    );
  }
}
