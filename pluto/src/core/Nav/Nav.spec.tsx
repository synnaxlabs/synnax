// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { MdHomeFilled } from "react-icons/md";
import { describe, expect, it } from "vitest";

import { Nav } from ".";

describe("Nav", () => {
  describe("Drawer", () => {
    it("should render an icon to the screen when we provide an item", () => {
      const props = Nav.useDrawer({
        items: [
          {
            key: "home",
            icon: <MdHomeFilled aria-label="home" />,
            content: <div>Home</div>,
          },
          {
            key: "space",
            icon: <MdHomeFilled aria-label="space" />,
            content: <div>Space</div>,
          },
        ],
      });
      const rendered = render(<Nav.Drawer location="left" {...props} />);
      expect(rendered.getByLabelText("home")).toBeTruthy();
      expect(rendered.getByLabelText("space")).toBeTruthy();
    });
    it("should render the correct content for the active key", async () => {
      const props = Nav.useDrawer({
        items: [
          {
            key: "space",
            icon: <MdHomeFilled aria-label="space" />,
            content: <div>SPACE!</div>,
          },
          {
            key: "something",
            icon: <MdHomeFilled aria-label="something" />,
            content: <div>something!</div>,
          },
        ],
      });
      const rendered = render(<Nav.Drawer location="left" {...props} />);
      const icon = rendered.getByLabelText("space");
      expect(await rendered.queryByText("SPACE!")).toBeFalsy();
      fireEvent.click(icon);
      expect(rendered.getByText("SPACE!")).toBeTruthy();
    });
    it("should render a horizontal drawer correctly", () => {
      const props = Nav.useDrawer({
        items: [
          {
            key: "space",
            icon: <MdHomeFilled aria-label="space" />,
            content: <div>SPACE!</div>,
          },
          {
            key: "something",
            icon: <MdHomeFilled aria-label="something" />,
            content: <div>something!</div>,
          },
        ],
      });
      const rendered = render(<Nav.Drawer location="bottom" {...props} />);
      expect(rendered.getByLabelText("something")).toBeTruthy();
      expect(rendered.getByLabelText("space")).toBeTruthy();
      expect(
        rendered.getByLabelText("space").parentElement?.parentElement?.className
      ).toContain("horizontal");
    });
  });
});
