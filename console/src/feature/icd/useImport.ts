// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { ICD } from "@synnaxlabs/pluto";

import { Runtime } from "@/platform/runtime";

const formatOf = (path: string): icd.ImportFormat =>
  icd.importFormatZ.parse(path.split(".").at(-1)?.toLowerCase());

/**
 * @returns a function that asks for a DBC, CSV, or XLSX file and imports it into the
 * ICD with the given key. The file's extension selects the format.
 */
export const useImport = (): ((key: icd.Key) => void) => {
  const handleError = Status.useErrorHandler();
  const { updateAsync } = ICD.useImport();
  return (key: icd.Key) =>
    handleError(async () => {
      const file = await Runtime.pickFiles({
        title: "Import ICD",
        extension: [...icd.IMPORT_FORMATS],
      });
      if (file == null) return;
      const data = await Runtime.toBytes(await file.read());
      await updateAsync({ key, format: formatOf(file.path), data });
    }, "Failed to import ICD");
};
