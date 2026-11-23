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

import { Button } from "@/button";

describe("CreateListItem", () => {
  it("should take the modifier of its size", () => {
    const { getByRole } = render(
      <Button.CreateListItem size="small">Add alert</Button.CreateListItem>,
    );
    const item = getByRole("button", { name: "Add alert" });
    expect(item.classList).toContain("lyra-create-list-item");
    expect(item.classList).toContain("lyra-create-list-item--small");
  });

  it("should be a small button at the small size", () => {
    const { getByRole } = render(
      <Button.CreateListItem size="small">Add alert</Button.CreateListItem>,
    );
    const item = getByRole("button", { name: "Add alert" });
    expect(item.classList).toContain("lyra--height-small");
    expect(item.classList).toContain("lyra-text--small");
  });

  it("should keep medium text at the large size", () => {
    const { getByRole } = render(
      <Button.CreateListItem size="large">Add alert</Button.CreateListItem>,
    );
    expect(getByRole("button", { name: "Add alert" }).classList).toContain(
      "lyra-text--p",
    );
  });

  it("should take the medium size by default", () => {
    const { getByRole } = render(
      <Button.CreateListItem>Add alert</Button.CreateListItem>,
    );
    expect(getByRole("button", { name: "Add alert" }).classList).toContain(
      "lyra-create-list-item--medium",
    );
  });

  it("should call onClick when clicked", () => {
    const onClick = vi.fn();
    const { getByRole } = render(
      <Button.CreateListItem onClick={onClick}>Add alert</Button.CreateListItem>,
    );
    fireEvent.click(getByRole("button", { name: "Add alert" }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
