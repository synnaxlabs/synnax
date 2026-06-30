// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { type FileIngesters } from "@/import/ingester";

const [Context, useContext] = context.create<FileIngesters>({
  displayName: "Import.Context",
  providerName: "Import.FileIngestersProvider",
});

export const useFileIngesters = (): FileIngesters =>
  useContext("Import.useFileIngesters");

export interface FileIngestersProviderProps extends PropsWithChildren {
  fileIngesters: FileIngesters;
}

export const FileIngestersProvider = ({
  fileIngesters,
  ...rest
}: FileIngestersProviderProps): ReactElement => (
  <Context value={fileIngesters} {...rest} />
);
