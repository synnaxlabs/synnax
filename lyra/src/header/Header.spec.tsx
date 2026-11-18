// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vitest } from "vitest";

import { Button } from "@/button";
import { Header } from "@/header";
import { Icon } from "@/icon";
import { Text } from "@/text";

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
  describe("Header.Group", () => {
    it("should call onOpenChange with the next state on a click", async () => {
      const onOpenChange = vitest.fn();
      const c = render(
        <Header.Group open onOpenChange={onOpenChange}>
          Plant
        </Header.Group>,
      );
      const toggle = c.getByRole("button", { name: "Plant" });
      expect(toggle.getAttribute("aria-expanded")).toBe("true");
      await userEvent.click(toggle);
      expect(onOpenChange).toHaveBeenCalledExactlyOnceWith(false);
    });

    it("should not open or close the group from a click in a title being edited", async () => {
      const onOpenChange = vitest.fn();
      const c = render(
        <Header.Group open onOpenChange={onOpenChange}>
          <Text.Editable id="title" value="Plant" onChange={vitest.fn()} />
        </Header.Group>,
      );
      act(() => Text.edit("title"));
      const title = await c.findByText("Plant");
      await waitFor(() => expect(title.getAttribute("contenteditable")).toBe("true"));
      await userEvent.click(title);
      expect(onOpenChange).not.toHaveBeenCalled();
      await userEvent.keyboard(" ");
      expect(onOpenChange).not.toHaveBeenCalled();
    });

    it("should not open or close the group from a click on an action", async () => {
      const onOpenChange = vitest.fn();
      const onCreate = vitest.fn();
      const c = render(
        <Header.Group
          open={false}
          onOpenChange={onOpenChange}
          actions={
            <Button.Button variant="text" aria-label="Create" onClick={onCreate}>
              <Icon.Add />
            </Button.Button>
          }
        >
          Plant
        </Header.Group>,
      );
      await userEvent.click(c.getByRole("button", { name: "Create" }));
      expect(onCreate).toHaveBeenCalledOnce();
      expect(onOpenChange).not.toHaveBeenCalled();
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
