// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { AiOutlineAim } from "react-icons/ai";
import { describe, expect, it } from "vitest";

import { Button } from ".";

describe("Button", () => {
  describe("Default", () => {
    it("should render a button with the provided text", () => {
      const c = render(<Button size="small">Hello</Button>);
      expect(c.getByText("Hello")).toBeTruthy();
    });
  });
  describe("IconOnly", () => {
    it("should render a button with the provided icon", () => {
      const c = render(
        <Button.IconOnly size="small">
          <AiOutlineAim aria-label="icon" />
        </Button.IconOnly>
      );
      expect(c.getByLabelText("icon")).toBeTruthy();
    });
  });
});
