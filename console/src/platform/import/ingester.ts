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

export interface FileIngesterContext {
  client: Synnax | null;
  projectKey: project.Key;
  /**
   * The name of the file the data was read from, extension included. The Core names
   * the resource after the file when the file's contents carry no name.
   */
  fileName: string;
}

/**
 * Creates the resource the data describes and returns its ID. Opening a tab for it
 * belongs to the caller, which decides where it lands.
 */
export interface FileIngester {
  (
    data: unknown,
    ctx: FileIngesterContext,
  ): void | ontology.ID | Promise<void | ontology.ID>;
}

interface BundleIngesterContext {
  client: Synnax | null;
  store: Store;
}

/**
 * Imports a zipped bundle read from a picked or dropped source. name is the source
 * archive or folder's name, the Core's fallback for naming the imported resource.
 */
export interface BundleIngester {
  (
    name: string,
    bundle: Uint8Array<ArrayBuffer>,
    ctx: BundleIngesterContext,
  ): Promise<void>;
}
