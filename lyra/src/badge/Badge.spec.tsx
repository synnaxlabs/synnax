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

import { Badge } from "@/badge";

describe("Badge", () => {
  it("should render its text", () => {
    const c = render(<Badge.Badge>Dev3</Badge.Badge>);
    expect(c.getByText("Dev3").classList).toContain("lyra-badge");
  });

  it("should tint the badge with the color of its status", () => {
    const c = render(<Badge.Badge status="error">Error</Badge.Badge>);
    expect(c.getByText("Error").classList).toContain("lyra--status-error");
  });

  it("should add no status color without a status", () => {
    const c = render(<Badge.Badge>Dev3</Badge.Badge>);
    expect(
      [...c.getByText("Dev3").classList].filter((cls) =>
        cls.startsWith("lyra--status"),
      ),
    ).toEqual([]);
  });

  it("should add a custom class to the badge", () => {
    const c = render(<Badge.Badge className="custom">Auto</Badge.Badge>);
    expect(c.getByText("Auto").classList).toContain("custom");
  });
});
