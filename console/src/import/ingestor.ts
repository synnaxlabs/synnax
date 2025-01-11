// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";

import { type Layout } from "@/layout";

interface FileIngestorContext {
  layout?: Partial<Layout.State>;
  placeLayout: Layout.Placer;
  store: Store;
}

export interface FileIngestor {
  (data: string, ctx: FileIngestorContext): void;
}

interface FileInfo {
  name: string;
  data: string;
}

interface DirectoryIngestorContext {
  client: Synnax | null;
  ingestors: Record<string, FileIngestor>;
  placeLayout: Layout.Placer;
  store: Store;
}

export interface DirectoryIngestor {
  (name: string, file: FileInfo[], ctx: DirectoryIngestorContext): Promise<void>;
}
