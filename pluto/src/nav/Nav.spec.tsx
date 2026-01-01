// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Nav } from "@/nav";

describe("Nav", () => {
  describe("Nav.Bar", () => {
    it("should render a navbar in the correct location", () => {
      const c = render(
        <Nav.Bar location="left">
          <Nav.Bar.End>
            <p>End</p>
          </Nav.Bar.End>
          <Nav.Bar.Start>
            <p>Start</p>
          </Nav.Bar.Start>
        </Nav.Bar>,
      );
      expect(c.getByText("Start")).toBeTruthy();
      expect(c.getByText("End")).toBeTruthy();
    });
  });
  describe("Nav.Drawer", () => {
    it("should show the correct item", () => {
      const items: Nav.DrawerItem[] = [
        { key: "1", content: <h1>Item 1 Content</h1> },
        { key: "2", content: <h1>Item 2 Content</h1> },
      ];
      const TestNavDrawer = (): ReactElement => {
        const props = Nav.useDrawer({ items, initialKey: "1" });
        return <Nav.Drawer {...props} />;
      };

      const c = render(<TestNavDrawer />);
      expect(c.getByText("Item 1 Content")).toBeTruthy();
      expect(c.queryByText("Item 2 Content")).toBeNull();
    });
  });
});
