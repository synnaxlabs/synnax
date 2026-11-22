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
import { describe, expect, it, vi } from "vitest";

import { Select } from "@/select";
import { Status } from "@/status";

const select = (status?: Status.Status): ReactElement => (
  <Select.Single
    visible
    data={["a", "b"]}
    virtual={false}
    onChange={vi.fn()}
    resourceName="result"
    status={status}
  >
    {({ itemKey }) => <div key={itemKey}>{`item ${itemKey}`}</div>}
  </Select.Single>
);

describe("Select.Single dialog body check", () => {
  it("should keep the list mounted, with the failure outside it, as the status changes", () => {
    const c = render(select());
    const list = c.getByRole("listbox");
    c.rerender(select(Status.create({ variant: "error", message: "Error" })));
    expect(c.getByRole("listbox")).toBe(list);
    expect(list.contains(c.getByText("Error"))).toBe(false);
    c.rerender(select(Status.create({ variant: "loading", message: "Loading" })));
    expect(c.getByRole("listbox")).toBe(list);
    c.rerender(select(Status.create({ variant: "error", message: "Error" })));
    expect(c.getByRole("listbox")).toBe(list);
  });
});
