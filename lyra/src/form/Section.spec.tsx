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

import { Form } from "@/form";

describe("Form.Sections", () => {
  it("should stack its sections by default", () => {
    const c = render(<Form.Sections data-testid="sections" />);
    const classes = c.getByTestId("sections").classList;
    expect(classes).toContain("pluto-form-sections");
    expect(classes).toContain("pluto--direction-y");
  });

  it("should lay its sections side by side when x is set", () => {
    const c = render(<Form.Sections x data-testid="sections" />);
    const classes = c.getByTestId("sections").classList;
    expect(classes).toContain("pluto--direction-x");
    expect(classes).not.toContain("pluto--direction-y");
  });
});

describe("Form.Section", () => {
  it("should render its title above its fields", () => {
    const c = render(
      <Form.Section title="Connection">
        <input aria-label="Host" />
      </Form.Section>,
    );
    const header = c.getByText("Connection").closest(".pluto-form-section__header");
    const body = c.getByLabelText("Host").closest(".pluto-form-section__body");
    expect(header).not.toBeNull();
    expect(body).not.toBeNull();
    expect(header?.parentElement).toBe(body?.parentElement);
  });

  it("should render its actions in the header", () => {
    const c = render(
      <Form.Section title="Channels" actions={<button>Add</button>}>
        <input aria-label="Name" />
      </Form.Section>,
    );
    expect(
      c.getByRole("button", { name: "Add" }).closest(".pluto-form-section__header"),
    ).not.toBeNull();
  });
});
