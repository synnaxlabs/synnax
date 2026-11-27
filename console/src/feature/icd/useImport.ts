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

import { Panel } from "@/platform/panel";
import { Runtime } from "@/platform/runtime";

interface Picked {
  /** The name of the file, without its extension. */
  name: string;
  format: icd.ImportFormat;
  data: Uint8Array;
}

/** Asks for a DBC, CSV, or XLSX file. The file's extension selects the format. */
const pick = async (): Promise<Picked | null> => {
  const file = await Runtime.pickFiles({
    title: "Import ICD",
    extension: [...icd.IMPORT_FORMATS],
  });
  if (file == null) return null;
  const dot = file.path.lastIndexOf(".");
  return {
    name: file.path.slice(0, dot),
    format: icd.importFormatZ.parse(file.path.slice(dot + 1).toLowerCase()),
    data: await Runtime.toBytes(await file.read()),
  };
};

/**
 * @returns a function that asks for a DBC, CSV, or XLSX file and imports it into the
 * ICD with the given key.
 */
export const useImport = (): ((key: icd.Key) => void) => {
  const handleError = Status.useErrorHandler();
  const { updateAsync } = ICD.useImport();
  return (key: icd.Key) =>
    handleError(async () => {
      const picked = await pick();
      if (picked == null) return;
      const { format, data } = picked;
      await updateAsync({ key, format, data });
    }, "Failed to import ICD");
};

/**
 * @returns a function that asks for a DBC, CSV, or XLSX file, creates an ICD named
 * after the file, and opens the ICD in a tab.
 */
export const useImportNew = (): (() => void) => {
  const handleError = Status.useErrorHandler();
  const openTab = Panel.useOpenTab();
  const { updateAsync } = ICD.useImportNew({
    afterSuccess: ({ data: { key } }) =>
      openTab({ variant: "resource", resource: icd.resourceID(key) }),
  });
  return () =>
    handleError(async () => {
      const picked = await pick();
      if (picked != null) await updateAsync(picked);
    }, "Failed to import ICD");
};
