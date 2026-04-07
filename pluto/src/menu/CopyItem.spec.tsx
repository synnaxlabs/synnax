// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Icon } from "@/icon";
import { Menu } from "@/menu";

describe("CopyItem", () => {
  const writeText = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers();
    Object.assign(navigator, {
      clipboard: { writeText },
    });
    writeText.mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
    writeText.mockReset();
  });

  it("should render children as the menu item content", () => {
    const c = render(
      <Menu.Menu>
        <Menu.CopyItem itemKey="copy" text="hello">
          <Icon.Python /> Copy Python code
        </Menu.CopyItem>
      </Menu.Menu>,
    );
    expect(c.getByText("Copy Python code")).toBeTruthy();
  });

  it("should copy text to the clipboard when clicked", async () => {
    const c = render(
      <Menu.Menu>
        <Menu.CopyItem itemKey="copy" text="copied content">Copy</Menu.CopyItem>
      </Menu.Menu>,
    );
    await act(async () => {
      fireEvent.click(c.getByText("Copy"));
    });
    expect(writeText).toHaveBeenCalledWith("copied content");
  });

  it("should copy the result of a function", async () => {
    const getText = vi.fn(() => "dynamic content");
    const c = render(
      <Menu.Menu>
        <Menu.CopyItem itemKey="copy" text={getText}>Copy</Menu.CopyItem>
      </Menu.Menu>,
    );
    await act(async () => {
      fireEvent.click(c.getByText("Copy"));
    });
    expect(getText).toHaveBeenCalled();
    expect(writeText).toHaveBeenCalledWith("dynamic content");
  });

  it("should copy the result of an async function", async () => {
    const getText = vi.fn(async () => "async content");
    const c = render(
      <Menu.Menu>
        <Menu.CopyItem itemKey="copy" text={getText}>Copy</Menu.CopyItem>
      </Menu.Menu>,
    );
    await act(async () => {
      fireEvent.click(c.getByText("Copy"));
    });
    expect(getText).toHaveBeenCalled();
    expect(writeText).toHaveBeenCalledWith("async content");
  });

  it("should call the menu context onClick with the itemKey", async () => {
    const onChange = vi.fn();
    const c = render(
      <Menu.Menu onChange={onChange}>
        <Menu.CopyItem itemKey="copyDiag" text="hello">
          Copy Diagnostics
        </Menu.CopyItem>
      </Menu.Menu>,
    );
    await act(async () => {
      fireEvent.click(c.getByText("Copy Diagnostics"));
    });
    expect(onChange).toHaveBeenCalledWith("copyDiag");
  });

  it("should apply the menu-item class", () => {
    const c = render(
      <Menu.Menu>
        <Menu.CopyItem itemKey="copy" text="hello">Copy</Menu.CopyItem>
      </Menu.Menu>,
    );
    expect(c.container.querySelector(".pluto-menu-item")).toBeTruthy();
  });
});
