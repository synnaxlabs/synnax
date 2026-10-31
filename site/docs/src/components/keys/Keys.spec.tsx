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

import { Keys } from "@/components/keys/Keys";

describe("Keys", () => {
  it("should hold the keys of macOS and of Windows", () => {
    const { container } = render(<Keys trigger={["Control", "G"]} />);
    const mac = container.querySelector(".docs-keys__mac");
    const pc = container.querySelector(".docs-keys__pc");
    expect(mac?.querySelector(".lyra-icon--keyboard-command")).not.toBeNull();
    expect(mac?.textContent).toBe("G");
    expect(pc?.textContent).toBe("CtrlG");
  });

  it("should give its own attributes to its root", () => {
    const { container } = render(
      <Keys trigger={["Escape"]} className="nav-keys" data-tooltip-keys />,
    );
    const root = container.firstElementChild;
    expect(root?.className).toBe("docs-keys nav-keys");
    expect(root?.hasAttribute("data-tooltip-keys")).toBe(true);
  });
});
