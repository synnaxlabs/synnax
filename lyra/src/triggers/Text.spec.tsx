// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Provider } from "@/triggers/Provider";
import { Text } from "@/triggers/Text";
import { type Trigger } from "@/triggers/triggers";

const TAB: Trigger = ["Control", "T"];

const reserving = (reserved: Trigger[]) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider reserved={reserved}>{children}</Provider>
  );
  return Wrapper;
};

describe("Text", () => {
  it("should render the trigger's keycaps", () => {
    const { container } = render(<Text trigger={TAB} />);
    expect(container.textContent).toContain("T");
  });

  it("should render the arrow keys as arrows", () => {
    const { container } = render(<Text trigger={["ArrowUp", "ArrowDown"]} />);
    expect(container.textContent).toBe("");
    expect(container.querySelector(".lyra-icon--keyboard-arrow-up")).not.toBeNull();
    expect(container.querySelector(".lyra-icon--keyboard-arrow-down")).not.toBeNull();
  });

  it("should render nothing for a reserved trigger", () => {
    const { container } = render(<Text trigger={TAB} />, {
      wrapper: reserving([["T", "Control"]]),
    });
    expect(container.innerHTML).toBe("");
  });

  it("should keep the label of a reserved trigger", () => {
    const { container } = render(<Text trigger={TAB}>New tab</Text>, {
      wrapper: reserving([TAB]),
    });
    expect(container.textContent).toBe("New tab");
  });

  it("should render a trigger the provider does not reserve", () => {
    const { container } = render(<Text trigger={["Control", "S"]} />, {
      wrapper: reserving([TAB]),
    });
    expect(container.textContent).toContain("S");
  });
});
