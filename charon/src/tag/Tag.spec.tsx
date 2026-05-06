// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Tag } from "@/tag";

describe("Tag", () => {
  it("should render a tag", () => {
    const c = render(<Tag.Tag>Test</Tag.Tag>);
    expect(c.getByText("Test")).toBeTruthy();
  });
  it("should render a tag wiht an icon", () => {
    const c = render(<Tag.Tag icon={<div>Icon</div>}>Test</Tag.Tag>);
    expect(c.getByText("Test")).toBeTruthy();
    expect(c.getByText("Icon")).toBeTruthy();
  });
  it("should render a close button if onClose is provided", () => {
    const close = vi.fn();
    const c = render(<Tag.Tag onClose={close}>Test</Tag.Tag>);
    const btn = c.getByLabelText("close");
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(close).toHaveBeenCalled();
  });
});
