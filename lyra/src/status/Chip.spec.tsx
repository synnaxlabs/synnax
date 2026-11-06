// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Status } from "@/status";

describe("Chip", () => {
  it("should render its word", () => {
    const c = render(<Status.Chip variant="success">Connected</Status.Chip>);
    expect(c.getByText("Connected").classList).toContain("lyra-status-chip");
  });

  it("should tint the chip with the variant color", () => {
    const c = render(<Status.Chip variant="error">Error</Status.Chip>);
    expect(c.getByText("Error").classList).toContain("lyra--status-error");
  });

  it("should add a custom class to the chip", () => {
    const c = render(
      <Status.Chip variant="warning" className="custom">
        Warning
      </Status.Chip>,
    );
    expect(c.getByText("Warning").classList).toContain("custom");
  });
});
