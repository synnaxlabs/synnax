// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { assertDefined } from "@synnaxlabs/x/testutil";
import { screen } from "@testing-library/react";

import { getInputItem } from "@/testutil";

/** @returns the hex box of the color control labeled with the given text. */
export const getHexInput = (label: string): HTMLInputElement => {
  const input = screen
    .getAllByLabelText(label)
    .find((el): el is HTMLInputElement => el instanceof HTMLInputElement);
  assertDefined(input);
  return input;
};

/** @returns the color swatches in the form item labeled with the given text. */
export const getSwatches = (label: string): HTMLElement[] =>
  Array.from(getInputItem(label).querySelectorAll<HTMLElement>(".lyra-color-swatch"));
