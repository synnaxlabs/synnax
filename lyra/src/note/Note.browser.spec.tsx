// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Note } from "@/note";
import { computed, renderThemed, resolve } from "@/styletest";
import { Text } from "@/text";

const VARIANTS = [
  ["info", "primary"],
  ["warning", "warning"],
  ["error", "error"],
] as const;

describe("Note", () => {
  it.each(VARIANTS)(
    "should color every text level in a %s note",
    async (variant, color) => {
      const { getByText } = await renderThemed(
        <Note.Note variant={variant}>
          <Text.Text level="p">Paragraph</Text.Text>
          <Text.Text level="small">Small</Text.Text>
        </Note.Note>,
      );
      const expected = resolve(`var(--pluto-${color}-text)`);
      for (const label of ["Paragraph", "Small"])
        expect(computed(getByText(label).element(), "color"), label).toBe(expected);
    },
  );
});
