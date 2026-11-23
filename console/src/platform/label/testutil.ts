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
import { onTestFinished } from "vitest";

/**
 * Returns the color of the dot beside the element, such as the dot of a label tag,
 * option, or filter chip, or null if it has none. It reads the first dot in the nearest
 * element around it that holds one.
 */
export const dotColorBeside = (element: HTMLElement): string | null => {
  for (let el: HTMLElement | null = element; el != null; el = el.parentElement) {
    const dot = el.querySelector("svg[color]");
    if (dot != null) return dot.getAttribute("color");
  }
  return null;
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
    color: { type: "fixed", value: "#000000" },
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
