// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";
import { type Pluto } from "@synnaxlabs/pluto";

import { type Layout } from "@/layout";

export interface File {
  data: unknown;
  name: string;
}

export interface FileIngesterContext {
  layout: Partial<Layout.State>;
  placeLayout: Layout.Placer;
  store: Pluto.FluxStore;
  client: Synnax | null;
}

export interface FileIngester {
  (data: unknown, ctx: FileIngesterContext): void;
}

export interface FileIngesters extends Record<string, FileIngester> {}

interface DirectoryIngesterContext {
  client: Synnax | null;
  fileIngesters: FileIngesters;
  placeLayout: Layout.Placer;
  store: Store;
  fluxStore: Pluto.FluxStore;
}

export interface DirectoryIngester {
  (name: string, files: File[], ctx: DirectoryIngesterContext): Promise<void>;
}
