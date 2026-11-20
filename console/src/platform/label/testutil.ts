// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { label, Synnax } from "@synnaxlabs/client";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { onTestFinished } from "vitest";

import { findDialogTrigger } from "@/testutil";

/**
 * Opens the mounted label select dialog, narrows the paginated list down to name via
 * the dialog's search input, and clicks the matching label option.
 */
export const searchAndClickLabel = async (name: string): Promise<void> => {
  fireEvent.click(await findDialogTrigger());
  await searchAndClickLabelOption(name);
};

/** Narrows the open label select dialog down to name and clicks the matching option. */
export const searchAndClickLabelOption = async (name: string): Promise<void> => {
  const search = await waitFor(() => screen.getByPlaceholderText("Search labels..."));
  fireEvent.change(search, { target: { value: name } });
  const option = await waitFor(() => screen.getByText(name));
  fireEvent.click(option);
};

/**
 * Creates a real label on the connected Core with a unique name. The overrides replace
 * those defaults. The label is deleted once the test finishes, so call it only inside a
 * test.
 */
export const createTestLabel = async (
  client: Synnax,
  overrides: Partial<label.New> = {},
): Promise<label.Label> => {
  const l = await client.labels.create({
    name: uniqueName("label"),
    color: "#000000",
    ...overrides,
  });
  onTestFinished(async () => await client.labels.delete(l.key));
  return l;
};

/**
 * Creates a unique label name for a label that the spec creates through the Console,
 * and deletes the labels with that name once the test finishes.
 */
export const createTestLabelName = (client: Synnax, prefix = "label"): string => {
  const name = uniqueName(prefix);
  onTestFinished(async () => {
    const labels = await client.labels.retrieve({ name: [name] });
    await client.labels.delete(labels.map((l) => l.key));
  });
  return name;
};
