// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FileTransport, type UploadBody } from "@synnaxlabs/freighter";

import { ontology } from "@/ontology";
import { project } from "@/project";

/**
 * The serialized wire formats a resource can be imported from or exported to. Today
 * only "JSON" is supported.
 */
export type Encoding = "JSON";

/**
 * Options shared by import and export. Carries the wire format of the resource and is
 * the extension point for any future per-call settings.
 */
export interface Options {
  /** The serialized format of the resource. */
  encoding: Encoding;
}

/** Options for a single import call. */
export interface ImportOptions extends Options {
  /**
   * The name of the file the resource was read from. When the file's contents carry no
   * top-level `name` field, the Core names the imported resource after the file, with
   * any trailing extension stripped. A `name` in the file always wins.
   */
  fileName: string;
  /**
   * The key of the project to create the imported resource under. The Core creates the
   * resource and its project relationship in a single transaction, so a parenting
   * failure rolls back the import.
   */
  project: project.Key;
}

/**
 * Imports and exports resources to and from a Synnax cluster as serialized bytes — the
 * same on-disk representation a user reads from or writes to a file. The bytes are
 * opaque to the client: nothing is parsed or transformed on the way through, so the
 * wire format is exactly the file's contents.
 *
 * Each call moves exactly one resource. When source is a ReadableStream or Blob, the
 * client streams the payload without buffering it in its own memory; an ArrayBufferView
 * or string is sent as-is (already in memory). Exports always stream back without
 * buffering. The cluster, however, currently materializes each payload server-side, so
 * a single import or export is bounded by the server's available memory.
 *
 * The client is environment-agnostic — the byte streams are standard Web Streams that
 * work in browsers, Node 18+, and Tauri webviews. Callers bridge the filesystem with
 * their own runtime's primitives:
 *
 * - Node: `import(Readable.toWeb(createReadStream(path)), opts)` and `(await export(id,
 *   opts)).pipeTo(Writable.toWeb(createWriteStream(path)))`.
 * - Browser: `import(fileInput.files[0], opts)` and pipe the export into a
 *   showSaveFilePicker writable (or buffer it with `new Response(stream).blob()`).
 * - Tauri/Console: read the picked file into a Blob to import, and hand the export
 *   stream to the Console's downloadStream helper.
 */
export class Client {
  private readonly fileTransport: FileTransport;

  constructor(file: FileTransport) {
    this.fileTransport = file;
  }

  /**
   * Imports a resource from the given serialized source and returns its new ontology
   * ID.
   *
   * @param source - the serialized resource (e.g. a file's contents). A ReadableStream
   * or Blob is streamed to the Core without buffering it in client memory; an
   * ArrayBufferView or string is sent as-is.
   * @param options - the import options, including the wire format of source, the
   * source file's name (used to name the resource when the file carries no name), and
   * the project to create the resource under.
   * @returns the new resource's ontology ID as stamped by the Core.
   */
  async import(
    source: UploadBody,
    { encoding, fileName, project: projectKey }: ImportOptions,
  ): Promise<ontology.ID> {
    return await this.fileTransport.upload(
      "/imex/import",
      source,
      {
        encoding,
        params: {
          file_name: fileName,
          parent: ontology.idToString(project.ontologyID(projectKey)),
        },
      },
      ontology.idZ,
    );
  }

  /**
   * Exports the resource identified by id as a byte stream. The caller pipes the stream
   * wherever it likes — a file on disk, a browser download, or an in-memory buffer via
   * `new Response(stream).json()` — without the client ever buffering the whole
   * payload.
   *
   * @param id - the ontology id of the resource to export.
   * @param options - the export options, including the desired wire format.
   * @returns the serialized resource as a stream of bytes.
   */
  async export(id: ontology.ID, options: Options): Promise<ReadableStream<Uint8Array>> {
    return await this.fileTransport.download("/imex/export", id, ontology.idZ, options);
  }
}
