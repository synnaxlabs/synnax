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

import { Breadcrumb } from "@/breadcrumb";

describe("Breadcrumb", () => {
  describe("Breadcrumb", () => {
    it("should render a breadcrumb with a single segment", () => {
      const c = render(
        <Breadcrumb.Breadcrumb>
          <Breadcrumb.Segment>Home</Breadcrumb.Segment>
        </Breadcrumb.Breadcrumb>,
      );
      expect(c.getByText("Home")).toBeTruthy();
    });
    it("should render a breadcrumb with multiple segments", () => {
      const c = render(
        <Breadcrumb.Breadcrumb>
          <Breadcrumb.Segment>Home</Breadcrumb.Segment>
          <Breadcrumb.Segment>Settings</Breadcrumb.Segment>
          <Breadcrumb.Segment>Profile</Breadcrumb.Segment>
        </Breadcrumb.Breadcrumb>,
      );
      expect(c.getByText("Home")).toBeTruthy();
      expect(c.getByText("Settings")).toBeTruthy();
      expect(c.getByText("Profile")).toBeTruthy();
      expect(c.queryAllByLabelText("pluto-icon--caret-right")).toHaveLength(2);
    });
  });
  describe("mapURLSegments", () => {
    it("should map URL segments correctly", () => {
      const segments = Breadcrumb.mapURLSegments(
        "home/settings/profile",
        ({ href, segment }) => (
          <Breadcrumb.Segment key={href} href={`/${href}`}>
            {segment}
          </Breadcrumb.Segment>
        ),
      );
      expect(segments).toHaveLength(3);

      const c = render(<Breadcrumb.Breadcrumb>{segments}</Breadcrumb.Breadcrumb>);
      const links = c.getAllByRole("link");
      expect(links).toHaveLength(3);
      expect(links[0].getAttribute("href")).toBe("/home");
      expect(links[1].getAttribute("href")).toBe("/home/settings");
      expect(links[2].getAttribute("href")).toBe("/home/settings/profile");
    });
  });
});
