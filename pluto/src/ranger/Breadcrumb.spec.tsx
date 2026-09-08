// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Ranger } from "@/ranger";
import { Text } from "@/text";

const STAGE_ICONS =
  ".pluto-icon--to-do, .pluto-icon--in-progress, .pluto-icon--completed";

describe("Breadcrumb", () => {
  describe("rendering", () => {
    it("should render the name", () => {
      const c = render(<Ranger.Breadcrumb name="Burst Test" />);
      expect(c.getByText("Burst Test")).toBeTruthy();
    });

    it("should render the parent name", () => {
      const c = render(
        <Ranger.Breadcrumb name="Burst Test" parent={{ name: "Campaign" }} />,
      );
      expect(c.getByText("Campaign")).toBeTruthy();
    });

    it("should not render the parent when showParent is false", () => {
      const c = render(
        <Ranger.Breadcrumb
          name="Burst Test"
          parent={{ name: "Campaign" }}
          showParent={false}
        />,
      );
      expect(c.queryByText("Campaign")).toBeNull();
    });

    it("should render the stage icon for the time range", () => {
      const now = TimeStamp.now();
      const c = render(
        <Ranger.Breadcrumb
          name="Burst Test"
          timeRange={{
            start: now.sub(TimeSpan.HOUR.mult(2)),
            end: now.sub(TimeSpan.HOUR),
          }}
        />,
      );
      expect(c.container.querySelector(".pluto-icon--completed")).toBeTruthy();
    });

    it("should not render a stage icon without a time range", () => {
      const c = render(<Ranger.Breadcrumb name="Burst Test" />);
      expect(c.container.querySelector(STAGE_ICONS)).toBeNull();
    });
  });

  describe("renaming", () => {
    it("should render the name as plain text without onRename", () => {
      const c = render(<Ranger.Breadcrumb name="Burst Test" nameID="range-name" />);
      expect(c.getByText("Burst Test").getAttribute("contenteditable")).toBeNull();
    });

    it("should render the name as an editable field with onRename", () => {
      const c = render(
        <Ranger.Breadcrumb name="Burst Test" nameID="range-name" onRename={vi.fn()} />,
      );
      expect(c.getByText("Burst Test").getAttribute("contenteditable")).toBe("false");
    });

    it("should not start an edit on double-click", () => {
      const c = render(
        <Ranger.Breadcrumb name="Burst Test" nameID="range-name" onRename={vi.fn()} />,
      );
      const text = c.getByText("Burst Test");
      fireEvent.dblClick(text);
      expect(text.getAttribute("contenteditable")).toBe("false");
    });

    it("should commit an edit started through Text.edit", () => {
      const onRename = vi.fn();
      const c = render(
        <Ranger.Breadcrumb name="Burst Test" nameID="range-name" onRename={onRename} />,
      );
      const text = c.getByText("Burst Test");
      act(() => Text.edit("range-name"));
      expect(text.getAttribute("contenteditable")).toBe("true");
      text.innerText = "Renamed";
      fireEvent.keyDown(text, { key: "Enter" });
      expect(onRename).toHaveBeenCalledWith("Renamed");
    });
  });
});
