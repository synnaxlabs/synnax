// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Accordion } from "@/accordion";

describe("Accordion", () => {
  const data: Accordion.Entry[] = [
    {
      key: "1",
      name: "Pane 1",
      content: <div>Pane 1 Content</div>,
      initialSize: 0.5,
    },
    {
      key: "2",
      name: "Pane 2",
      content: <div>Pane 2 Content</div>,
      initialSize: 0.5,
    },
  ];
  it("should render two panes with the correct sizes", () => {
    const c = render(<Accordion.Accordion data={data} />);
    expect(c.getByText("Pane 1")).toBeTruthy();
    const parent = c.getByText("Pane 1 Content").parentElement;
    expect(parent).toBeTruthy();
    // Intentionally a little bit off
    expect(parent?.style.height).toBe("50.1%");
  });
  it("should render the correct expand/contract icons", () => {
    const c = render(<Accordion.Accordion data={data} />);
    expect(c.queryAllByLabelText("contract")).toHaveLength(2);
    expect(c.queryAllByLabelText("expand")).toHaveLength(0);
  });
});
