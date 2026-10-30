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

import { Input } from "@/input";

describe("Input.Effect", () => {
  it("should keep showing its last content while it hides", () => {
    const c = render(<Input.Effect>Moves the start</Input.Effect>);
    c.rerender(<Input.Effect>Moves the end</Input.Effect>);
    c.rerender(<Input.Effect>{null}</Input.Effect>);
    const effect = c.container.firstElementChild;
    expect(effect?.getAttribute("aria-hidden")).toBe("true");
    expect(effect?.textContent).toBe("Moves the end");
  });
});
