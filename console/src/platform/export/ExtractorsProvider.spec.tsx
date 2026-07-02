// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Export } from "@/platform/export";

const Consumer = (): ReactElement => {
  const extractors = Export.useExtractors();
  return <span>{Object.keys(extractors).join(",")}</span>;
};
Consumer.displayName = "Consumer";

describe("Export.ExtractorsProvider", () => {
  it("provides the extractors registry to descendants", () => {
    const extractors = { log: vi.fn(), table: vi.fn() };
    render(
      <Export.ExtractorsProvider extractors={extractors}>
        <Consumer />
      </Export.ExtractorsProvider>,
    );
    expect(screen.getByText("log,table")).toBeTruthy();
  });

  it("throws when useExtractors is called outside of a provider", () => {
    expect(() => render(<Consumer />)).toThrow();
  });
});
