// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { installDirectoryPicker } from "@/testutil";

export interface InstallPickedDirectoryOptions {
  /**
   * Whether the export subdirectory already exists in the picked directory. When
   * false, the first lookup rejects with NotFoundError so the export proceeds
   * without a replace confirmation.
   */
  exists?: boolean;
}

/**
 * Installs a fake FS Access directory picker whose picked directory records every
 * file write into the returned map, keyed by file name. The fake handles cover the
 * minimal surface the project export path touches; jsdom cannot construct real
 * FileSystemDirectoryHandles, so the sanctioned cast lives here.
 */
export const installPickedDirectory = ({
  exists = true,
}: InstallPickedDirectoryOptions = {}): Map<string, string> => {
  const writes = new Map<string, string>();
  const subHandle = {
    getFileHandle: async (name: string) => ({
      createWritable: async () => ({
        write: async (data: string) => void writes.set(name, data),
        close: async () => {},
      }),
    }),
  };
  let firstLookup = true;
  const root = {
    name: "Downloads",
    getDirectoryHandle: async () => {
      if (!exists && firstLookup) {
        firstLookup = false;
        throw new DOMException("missing", "NotFoundError");
      }
      return subHandle;
    },
  };
  installDirectoryPicker(async () => root as unknown as FileSystemDirectoryHandle);
  return writes;
};
