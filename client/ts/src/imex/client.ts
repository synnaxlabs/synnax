// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FileTransport, type UploadBody } from "@synnaxlabs/freighter";
import { z } from "zod";

import { group } from "@/group";
import { resource } from "@/resource";

/**
 * The serialized wire formats a resource can be imported from or exported to. Today
 * only "JSON" is supported.
 */
export const encodingZ = z.enum(["JSON"]);

/** The serialized wire format of an imported or exported resource. */
export type Encoding = z.infer<typeof encodingZ>;

/**
 * Options shared by import and export. Carries the wire format of the resource and is
 * the extension point for any future per-call settings.
 */
export interface Options {
  /** The serialized format of the resource. */
  encoding: Encoding;
}

/** The standard options for a JSON import or export. */
export const JSON_OPTIONS: Options = { encoding: "JSON" };

/** The wire shape of the per-import request params. */
const importParamsZ = z.object({
  fileName: z.string().min(1),
  parent: resource.idZ.transform((id) => resource.idToString(id)).optional(),
  group: group.keyZ.optional(),
});

/** Options for a single import call. */
export interface ImportOptions extends Options {
  /**
   * The name of the file the resource was read from. When the file's contents carry no
   * top-level `name` field, the Core names the imported resource after the file, with
   * any trailing extension stripped. A `name` in the file always wins.
   */
  fileName: string;
  /**
   * The resource to create the imported resource under, such as the project of a
   * project item. The Core creates the resource with its parent in a single
   * transaction, so a parenting failure rolls back the import. Symbols take no parent.
   */
  parent?: resource.ID;
  /** The group to put the imported resource in, if any. */
  group?: group.Key;
}

/**
 * Imports and exports resources to and from a Synnax cluster as serialized bytes — the
 * same on-disk representation a user reads from or writes to a file. The bytes are
 * opaque to the client: nothing is parsed or transformed on the way through, so the
 * wire format is exactly the file's contents.
 *
 * Each call moves exactly one resource. When source is a Blob, the client never holds
 * the payload in its own memory; bytes are sent as-is (already in memory). Exports
 * always stream back without buffering. The Core, however, currently materializes each
 * payload, so a single import or export is bounded by its available memory.
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
 *   stream to the Console's download hook.
 */
export interface ClientConfig {
  file: FileTransport;
}

export class Client {
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    this.cfg = cfg;
  }

  /**
   * Imports a resource from the given serialized source and returns its new resource
   * ID.
   * @param source - the serialized resource (e.g. a file's contents). A Blob is sent
   * from wherever the engine holds it, so a file never enters client memory; bytes are
   * sent as-is.
   * @param options - the import options, including the wire format of source, the
   * source file's name (used to name the resource when the file carries no name), and
   * the parent resource to create the resource under, and the group to put it in.
   * @returns the new resource's resource ID as stamped by the Core.
   */
  async import(
    source: UploadBody,
    { encoding, ...params }: ImportOptions,
  ): Promise<resource.ID> {
    return await this.cfg.file.upload(
      "/imex/import",
      source,
      { encoding, params, paramsSchema: importParamsZ },
      resource.idZ,
    );
  }

  /**
   * Exports the resource identified by id as a byte stream. The caller pipes the stream
   * wherever it likes — a file on disk, a browser download, or an in-memory buffer via
   * `new Response(stream).json()` — without the client ever buffering the whole
   * payload.
   * @param options - the export options, including the desired wire format.
   * @returns the serialized resource as a stream of bytes.
   */
  async export(id: resource.ID, options: Options): Promise<ReadableStream<Uint8Array>> {
    return await this.cfg.file.download("/imex/export", id, resource.idZ, options);
  }
}
