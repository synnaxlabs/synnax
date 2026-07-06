// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { vi } from "vitest";

import { type dataTransferItem } from "@/platform/import/dataTransferItem";
import { type FileIngesterContext } from "@/platform/import/ingester";
import { type Layout } from "@/platform/layout";
import { Session } from "@/session";
import { createTestFluxStore, createTestStore } from "@/testutil";

export type DataTransferItemContext = Parameters<typeof dataTransferItem>[1];

/**
 * Builds a real FileIngesterContext: a live flux store, a spy placer injected via DI,
 * and a null client. Merge overrides over it for spec-specific fields.
 */
export const createFileIngesterContext = (
  overrides: Partial<FileIngesterContext> = {},
): FileIngesterContext => ({
  layout: { name: "test" },
  placeLayout: vi.fn<Layout.Placer>(),
  store: createTestFluxStore(),
  client: null,
  projectKey: "project-1",
  ...overrides,
});

/**
 * Builds a real DataTransferItemContext: the full console redux store, a live flux
 * store, spy ingest callbacks injected via DI, and a null client.
 */
export const createDataTransferItemContext = async (
  overrides: Partial<DataTransferItemContext> = {},
): Promise<DataTransferItemContext> => ({
  client: null,
  fileIngesters: {},
  ingestDirectory: vi.fn(),
  layout: {},
  placeLayout: vi.fn<Layout.Placer>(),
  store: await createTestStore({
    preloadedState: {
      [Session.Project.SLICE_NAME]: {
        ...Session.Project.ZERO_SLICE_STATE,
        selected: "project-1",
      },
    },
  }),
  fluxStore: createTestFluxStore(),
  ...overrides,
});

export interface FakeDataTransferItemArgs {
  /** The item kind; anything other than "file" makes the import path reject it. */
  kind?: string;
  /** The FileSystemEntry-shaped value webkitGetAsEntry returns. */
  entry?: unknown;
  /** The file getAsFile returns. */
  file?: File | null;
}

/**
 * Builds the minimal DataTransferItem surface the import path reads. jsdom cannot
 * construct a DataTransferItem, so the single sanctioned cast to the DOM type lives
 * here.
 */
export const fakeDataTransferItem = ({
  kind = "file",
  entry = null,
  file = null,
}: FakeDataTransferItemArgs = {}): DataTransferItem =>
  ({
    kind,
    webkitGetAsEntry: () => entry,
    getAsFile: () => file,
  }) as unknown as DataTransferItem;
