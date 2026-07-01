// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Extractor } from "@/feature/export/extractor";
import { Runtime } from "@/primitive/runtime";
import { Session } from "@/session";

const FILTERS: Runtime.FileFilter[] = [{ name: "JSON", extensions: ["json"] }];

export const use = (extract: Extractor, type: string): ((key: string) => void) => {
  const client = Synnax.use();
  const store = Session.useStore();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  return useCallback(
    (key: string) => {
      let name: string | undefined;
      handleError(
        async () => {
          const extractorReturn = await extract(key, { store, client });
          name = extractorReturn.name;
          const location = await Runtime.saveFile({
            title: `Export ${name}`,
            defaultName: `${name}.json`,
            filters: FILTERS,
            contents: extractorReturn.data,
          });
          if (location == null) return;
          addStatus({
            variant: "success",
            message: `Exported ${name ?? type} to ${location}`,
          });
        },
        `Failed to export ${name ?? type}`,
      );
    },
    [client, store, handleError, extract, type],
  );
};
