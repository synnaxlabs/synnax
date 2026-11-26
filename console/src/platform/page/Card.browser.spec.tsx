// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Page } from "@/platform/page";
import { renderWithConsole } from "@/testutil";

describe("Page.Card", () => {
  it("should drop the inner padding of a flush card", async () => {
    await renderWithConsole(
      <>
        <Page.Card title="Padded">padded</Page.Card>
        <Page.Card title="Flush" flush>
          flush
        </Page.Card>
      </>,
    );
    const FLUSH = "console-page__card--flush";
    expect(screen.getByText("padded").classList.contains(FLUSH)).toBe(false);
    expect(screen.getByText("flush").classList.contains(FLUSH)).toBe(true);
  });
});
