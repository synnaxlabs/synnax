// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { renderThemed, treatment } from "@/styletest";
import { Telem } from "@/telem";

const STAMP = TimeStamp.now().sub(TimeSpan.days(2));

describe("TimeSpanSince", () => {
  it("should draw the elapsed time with tabular digits", async () => {
    const { getByText } = await renderThemed(
      <Telem.Text.TimeSpanSince>{STAMP}</Telem.Text.TimeSpanSince>,
    );
    expect(treatment(getByText(/2d/).element())).toBe("numeric");
  });

  it("should take another variant", async () => {
    const { getByText } = await renderThemed(
      <Telem.Text.TimeSpanSince variant="prose">{STAMP}</Telem.Text.TimeSpanSince>,
    );
    expect(treatment(getByText(/2d/).element())).toBe("prose");
  });
});
