// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type ontology, type project, type Synnax } from "@synnaxlabs/client";

import { type Panel } from "@/platform/panel";

export interface File {
  data: unknown;
  name: string;
}

export interface FileIngesterContext {
  /** name is the fallback component name derived from the imported file. */
  name?: string;
  openTab: Panel.OpenTab;
  client: Synnax | null;
  projectKey: project.Key;
  /**
   * The name of the file the data was read from, extension included. Server-side
   * ingesters forward it so the Core can name the resource after the file when the
   * file's contents carry no name.
   */
  fileName: string;
}

export interface FileIngester {
  (
    data: unknown,
    ctx: FileIngesterContext,
  ): void | ontology.ID | Promise<void | ontology.ID>;
}

export interface FileIngesters extends Record<string, FileIngester> {}

interface DirectoryIngesterContext {
  client: Synnax | null;
  fileIngesters: FileIngesters;
  openTab: Panel.OpenTab;
  store: Store;
}

export interface DirectoryIngester {
  (name: string, files: File[], ctx: DirectoryIngesterContext): Promise<void>;
}
