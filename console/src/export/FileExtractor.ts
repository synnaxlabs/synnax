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

export interface FileExtractorContext {
  store: Store;
  client: Synnax | null;
}

export interface FileExtractorReturn {
  file: string;
  name: string;
}

export type FileExtractor = (
  key: string,
  context: FileExtractorContext,
) => Promise<FileExtractorReturn>;
