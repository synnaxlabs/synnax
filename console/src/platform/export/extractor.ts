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

export interface ExtractorContext {
  client: Synnax | null;
  store: Store;
}

export interface File {
  data: string;
  name: string;
}

// Extractor is a function that extracts a visualization or other object with the given
// key into the info for a file. The extractor interface should be defined for each
// service that needs to export data. The file info can then be stored in a file system
// and imported again using the importer interface.
export interface Extractor {
  (key: string, ctx: ExtractorContext): Promise<File>;
}

export interface Extractors extends Record<string, Extractor> {}
