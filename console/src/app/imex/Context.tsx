// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren } from "react";

import { Task } from "@/feature/task";
import { Import } from "@/platform/import";

// Only task configs are ingested client-side; every other file type is streamed to the
// server through Import.ingestServer.
const FILE_INGESTERS: Import.FileIngesters = { ...Task.FILE_INGESTERS };

export interface ContextProps extends PropsWithChildren<{}> {}

export const Context = ({ children }: ContextProps) => (
  <Import.FileIngestersProvider fileIngesters={FILE_INGESTERS}>
    {children}
  </Import.FileIngestersProvider>
);
