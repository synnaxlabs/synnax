// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Breadcrumb } from "@/breadcrumb";

describe("Breadcrumb", () => {
  describe("Breadcrumb", () => {
    it("should render a breadcrumb with a single segment", () => {
      const c = render(<Breadcrumb.Breadcrumb>{["Home"]}</Breadcrumb.Breadcrumb>);
      expect(c.getByText("Home")).toBeTruthy();
    });
    it("should render a breadcrumb with multiple segments", () => {
      const c = render(
        <Breadcrumb.Breadcrumb>
          {["Home", "Settings", "Profile"]}
        </Breadcrumb.Breadcrumb>,
      );
      expect(c.getByText("Home")).toBeTruthy();
      expect(c.getByText("Settings")).toBeTruthy();
      expect(c.getByText("Profile")).toBeTruthy();
      expect(c.getAllByLabelText("synnax-icon-caret-right")).toHaveLength(2);
    });
  });
  describe("URL", () => {
    it("should render a breadcrumb multiple segments", () => {
      const c = render(<Breadcrumb.URL url="home/settings/profile" />);
      expect(c.getByText("Home")).toBeTruthy();
      expect(c.getByText("Settings")).toBeTruthy();
      expect(c.getByText("Profile")).toBeTruthy();
      expect(c.getAllByLabelText("synnax-icon-caret-right")).toHaveLength(3);
      const home = c.getByText("Home");
      expect(home.getAttribute("href")).toBe("/home");
      const settings = c.getByText("Settings");
      expect(settings.getAttribute("href")).toBe("/home/settings");
      const profile = c.getByText("Profile");
      expect(profile.getAttribute("href")).toBe("/home/settings/profile");
    });
  });
});
