// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vitest } from "vitest";

import { Header } from ".";

describe("Header", () => {
  describe("Header.Title", () => {
    it("should render a header", () => {
      const c = render(
        <Header level="h1">
          <Header.Title>Header</Header.Title>
        </Header>
      );
      expect(c.getByText("Header")).toBeTruthy();
    });
  });
  describe("Header.Actions", () => {
    it("should render a header with action", async () => {
      const onClick = vitest.fn();
      const c = render(
        <Header>
          <Header.Title>Header</Header.Title>
          <Header.Actions>
            {[
              {
                onClick,
                children: <Icon.Add aria-label="add" />,
              },
            ]}
          </Header.Actions>
        </Header>
      );
      expect(c.getByText("Header")).toBeTruthy();
      const add = c.getByLabelText("add");
      expect(add).toBeTruthy();
      await userEvent.click(add);
      expect(onClick).toHaveBeenCalled();
    });
  });
  describe("Header.Title.Button", () => {
    it("should render a header with button as its title", async () => {
      const onClick = vitest.fn();
      const c = render(
        <Header>
          <Header.Title.Button onClick={onClick}>Header</Header.Title.Button>
        </Header>
      );
      expect(c.getByText("Header")).toBeTruthy();
      const button = c.getByText("Header");
      expect(button).toBeTruthy();
      await userEvent.click(button);
      expect(onClick).toHaveBeenCalled();
    });
  });
});
