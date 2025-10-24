// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";

import { type Layout } from "@/layout";

export interface File {
  data: unknown;
  name: string;
}

interface FileIngestorContext {
  layout: Partial<Layout.State>;
  placeLayout: Layout.Placer;
  store: Store;
}

export interface FileIngestor {
  (data: unknown, ctx: FileIngestorContext): void;
}

export interface FileIngestors extends Record<string, FileIngestor> {}

interface DirectoryIngestorContext {
  client: Synnax | null;
  fileIngestors: FileIngestors;
  placeLayout: Layout.Placer;
  store: Store;
}

export interface DirectoryIngestor {
  (name: string, files: File[], ctx: DirectoryIngestorContext): Promise<void>;
}
