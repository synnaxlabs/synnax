// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  certificate,
  type rack,
  status,
  type Synnax,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";
import { z } from "zod";

/** The type of the task that lists the files of a rack. */
export const TASK_TYPE = "file_browser";

const LIST_COMMAND_TYPE = "list";

const LIST_TIMEOUT = TimeSpan.seconds(10);

const entryZ = z.object({
  name: z.string(),
  /** The absolute path of the entry on the rack. */
  path: z.string(),
  kind: z.enum(["directory", "file"]),
  /** Unset for an entry at the top or in the secrets folder. */
  size: z.number().optional(),
  modified: TimeStamp.z.optional(),
  /** Set only for a certificate file. */
  certificate: certificate.detailsZ.optional(),
});

/** A file or folder on a rack. */
export interface Entry extends z.infer<typeof entryZ> {}

const listingZ = z.object({
  /** The listed folder, or empty for the top. */
  path: z.string(),
  /** The folder above the listed one, or empty at a root. */
  parent: z.string(),
  /** The secrets folder of the rack, or empty when it has none. */
  secretsDir: z.string(),
  /** Whether the listed folder is the secrets folder or inside it. */
  secret: z.boolean(),
  /** Whether the rack stopped at a limit before it listed every entry. */
  truncated: z.boolean(),
  entries: entryZ.array(),
});

/** The entries of a folder on a rack. */
export interface Listing extends z.infer<typeof listingZ> {}

/**
 * Lists a folder on a rack. An empty path lists the roots, the home folder, and the
 * secrets folder.
 * @throws {Error} if the rack has no file browser, or if the rack fails to list the
 * folder.
 */
export const list = async (
  client: Synnax,
  rack: rack.Key,
  path: string = "",
): Promise<Listing> => {
  const browser = await client.tasks.retrieve({ type: TASK_TYPE, rack, one: true });
  const res = await browser.executeCommandSync({
    type: LIST_COMMAND_TYPE,
    timeout: LIST_TIMEOUT,
    args: { path },
  });
  if (res.variant === "error") throw status.toError(res);
  return listingZ.parse(res.details?.data);
};
