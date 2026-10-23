// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vitest } from "vitest";

import { Button } from "@/button";
import { Header } from "@/header";
import { Icon } from "@/icon";

describe("Header", () => {
  describe("Header.Title", () => {
    it("should render a header", () => {
      const c = render(
        <Header.Header level="h1">
          <Header.Title>Header</Header.Title>
        </Header.Header>,
      );
      expect(c.getByText("Header")).toBeTruthy();
    });
  });
  describe("Header.Actions", () => {
    it("should render a header with action", async () => {
      const onClick = vitest.fn();
      const c = render(
        <Header.Header>
          <Header.Title>Header</Header.Title>
          <Header.Actions>
            <Button.Button onClick={onClick} variant="text">
              <Icon.Add aria-label="add" />
            </Button.Button>
          </Header.Actions>
        </Header.Header>,
      );
      expect(c.getByText("Header")).toBeTruthy();
      const add = c.getByLabelText("add");
      expect(add).toBeTruthy();
      await userEvent.click(add);
      expect(onClick).toHaveBeenCalled();
    });
  });
  describe("Header.Title.Use", () => {
    it("should render a header with button as its title", async () => {
      const onClick = vitest.fn();
      const c = render(
        <Header.Header>
          <Header.ButtonTitle onClick={onClick}>Header</Header.ButtonTitle>
        </Header.Header>,
      );
      expect(c.getByText("Header")).toBeTruthy();
      const button = c.getByText("Header");
      expect(button).toBeTruthy();
      await userEvent.click(button);
      expect(onClick).toHaveBeenCalled();
    });
  });
});
