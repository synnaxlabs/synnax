// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
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
});
