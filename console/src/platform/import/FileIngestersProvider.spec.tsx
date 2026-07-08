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

import { Import } from "@/platform/import";

const Consumer = (): ReactElement => {
  const ingesters = Import.useFileIngesters();
  return <span>{Object.keys(ingesters).join(",")}</span>;
};
Consumer.displayName = "Consumer";

describe("Import.FileIngestersProvider", () => {
  it("provides the file ingesters registry to descendants", () => {
    const fileIngesters = { log: vi.fn(), table: vi.fn() };
    render(
      <Import.FileIngestersProvider fileIngesters={fileIngesters}>
        <Consumer />
      </Import.FileIngestersProvider>,
    );
    expect(screen.getByText("log,table")).toBeTruthy();
  });

  it("throws when useFileIngesters is called outside of a provider", () => {
    expect(() => render(<Consumer />)).toThrow();
  });
});
