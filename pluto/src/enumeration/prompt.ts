// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/lyra/context";
import { id } from "@synnaxlabs/x";

/** @returns the entries a new enumeration starts with: one blank entry to fill in. */
export const createEntries = (): enumeration.Entry[] => [
  { key: id.create(), name: "", value: 0, abnormal: false },
];

export interface CreatePromptParams {
  /** The name that the field starts with. */
  name: string;
  /** The entries that the new enumeration starts with. */
  entries: enumeration.Entry[];
}

/**
 * Asks the user for the name and entries of a new enumeration, then creates it.
 *
 * @returns the key of the new enumeration, or null when the user cancels.
 */
export type CreatePrompt = (
  params: CreatePromptParams,
) => Promise<enumeration.Key | null>;

const [Context, useContext] = context.create<CreatePrompt>({
  displayName: "Enumeration.CreatePromptContext",
  providerName: "Enumeration.CreatePromptProvider",
});

/** CreatePromptProvider gives the forms below it the prompt for a new enumeration. */
export const CreatePromptProvider = Context;

/** @returns the prompt for a new enumeration. Throws outside a CreatePromptProvider. */
export const useCreatePrompt = (): CreatePrompt =>
  useContext("Enumeration.useCreatePrompt");
