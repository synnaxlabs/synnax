// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Library } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Runtime } from "@/platform/runtime";

const formatOf = (path: string): library.ImportFormat =>
  library.importFormatZ.parse(path.split(".").at(-1)?.toLowerCase());

export interface UseImportParams {
  /** Called with the library the Core holds after the import. */
  onImport?: (imported: library.Library) => void;
}

/**
 * @returns a function that asks for a DBC, CSV, or XLSX file and imports it into the
 * library with the given key. The file's extension selects the format.
 */
export const useImport = ({ onImport }: UseImportParams = {}): ((
  key: library.Key,
) => void) => {
  const handleError = Status.useErrorHandler();
  const { updateAsync } = Library.useImport({
    afterSuccess: ({ data }) => onImport?.(data),
  });
  return useCallback(
    (key: library.Key) =>
      handleError(async () => {
        const file = await Runtime.pickFiles({
          title: "Import library",
          extension: [...library.IMPORT_FORMATS],
        });
        if (file == null) return;
        const data = await Runtime.toBytes(await file.read());
        await updateAsync({ key, format: formatOf(file.path), data });
      }, "Failed to import library"),
    [handleError, updateAsync],
  );
};
