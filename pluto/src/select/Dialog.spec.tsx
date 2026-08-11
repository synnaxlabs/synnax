// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { Select } from "@/select";

describe("Select.Dialog", () => {
  it("should render the default empty content when the list has no data", () => {
    const onChange = vi.fn();
    const c = render(
      <Dialog.Frame visible>
        <Select.Frame data={[]} onChange={onChange}>
          <Select.Dialog resourceName="result">{() => <div>Hello</div>}</Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>,
    );
    expect(c.getByText("No results found")).toBeTruthy();
  });

  it("should render status content when the status is not success", () => {
    const onChange = vi.fn();
    const c = render(
      <Dialog.Frame visible>
        <Select.Frame data={[]} onChange={onChange}>
          <Select.Dialog
            resourceName="result"
            status={status.create({ variant: "error", message: "Error" })}
          >
            {() => <div>Hello</div>}
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>,
    );
    expect(c.getByText("Error")).toBeTruthy();
  });

  it("should accept a custom empty content", () => {
    const onChange = vi.fn();
    const c = render(
      <Dialog.Frame visible>
        <Select.Frame data={[]} onChange={onChange}>
          <Select.Dialog resourceName="result" emptyContent={<div>Hello</div>}>
            {() => <div>Hello</div>}
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>,
    );
    expect(c.getByText("Hello")).toBeTruthy();
  });

  it("should accept a string as empty content", () => {
    const onChange = vi.fn();
    const c = render(
      <Dialog.Frame visible>
        <Select.Frame data={[]} onChange={onChange}>
          <Select.Dialog resourceName="result" emptyContent="Hello">
            {() => <div>Hello</div>}
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>,
    );
    expect(c.getByText("Hello")).toBeTruthy();
  });

  describe("footer", () => {
    const renderDialog = (footer?: ReactNode) =>
      render(
        <Dialog.Frame visible>
          <Select.Frame data={[]} onChange={vi.fn()}>
            <Select.Dialog resourceName="result" footer={footer}>
              {() => <div>Hello</div>}
            </Select.Dialog>
          </Select.Frame>
        </Dialog.Frame>,
      );

    it("should render the footer", () => {
      const c = renderDialog(<button type="button">Create</button>);
      expect(c.getByText("Create")).toBeTruthy();
    });

    it("should keep the footer outside the scrolling list", () => {
      const c = renderDialog(<button type="button">Create</button>);
      const items = c.baseElement.querySelector(".pluto-list__items");
      expect(items).not.toBeNull();
      expect(items?.contains(c.getByText("Create"))).toBe(false);
    });

    it("should move the border onto the wrapper so the footer sits inside it", () => {
      const c = renderDialog(<button type="button">Create</button>);
      const body = c.baseElement.querySelector(".pluto-select__body");
      expect(body).not.toBeNull();
      expect(body?.contains(c.getByText("Create"))).toBe(true);
      expect(body?.className).toContain("pluto--bordered");
      expect(
        c.baseElement.querySelector(".pluto-list__items")?.className,
      ).not.toContain("pluto--bordered");
    });

    it("should leave the border on the list when there is no footer", () => {
      const c = renderDialog();
      expect(c.baseElement.querySelector(".pluto-select__body")).toBeNull();
      expect(c.baseElement.querySelector(".pluto-list__items")?.className).toContain(
        "pluto--bordered",
      );
    });
  });
});
