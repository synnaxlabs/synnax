// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { render, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { createConfig, REGISTRY, type Variant } from "@/schematic/node/registry";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const SynnaxWrapper = createSynnaxWrapper({ client: null });

const createFormWrapper = (variant: Variant) => {
  const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
    const methods = Form.use<typeof schematic.nodeConfigZ>({
      values: createConfig({ variant }),
      schema: schematic.nodeConfigZ,
    });
    return (
      <SynnaxWrapper>
        <Form.Form<typeof schematic.nodeConfigZ> {...methods}>{children}</Form.Form>
      </SynnaxWrapper>
    );
  };
  return FormWrapper;
};

describe("symbol forms that wrap a shared form", () => {
  it("should pass the selected tab through the tank form", () => {
    const { Form: TankForm } = REGISTRY.tank;
    render(<TankForm tab="fill" onTabChange={vi.fn()} />, {
      wrapper: createFormWrapper("tank"),
    });
    expect(screen.getByRole("tab", { name: "Fill" }).ariaSelected).toBe("true");
  });

  it("should pass the selected tab and actions through the switch form", () => {
    const { Form: SwitchForm } = REGISTRY.switch;
    render(
      <SwitchForm tab="style" onTabChange={vi.fn()} actions={<button>Swap</button>} />,
      { wrapper: createFormWrapper("switch") },
    );
    expect(screen.getByRole("tab", { name: "Style" }).ariaSelected).toBe("true");
    expect(screen.getByRole("button", { name: "Swap" })).toBeDefined();
  });
});
