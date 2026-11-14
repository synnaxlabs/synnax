// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { mockGeometry } from "@synnaxlabs/lyra/testutil";
import { id } from "@synnaxlabs/x";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { Role } from "@/access/role";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const BUILT_IN_ICONS: Record<string, string> = {
  Owner: "settings",
  Engineer: "safety",
  Operator: "control",
  Viewer: "visible",
};

const ownerKey = async (): Promise<access.role.Key> => {
  const roles = await client.access.roles.retrieve({});
  const owner = roles.find((r) => r.name === "Owner");
  if (owner == null) throw new Error("no Owner role");
  return owner.key;
};

describe("Role.Select", () => {
  // Without a box, the dropdown's virtual list shows no items.
  beforeAll(() => mockGeometry(100, 100));
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const open = async (props: Partial<Role.SelectProps> = {}): Promise<void> => {
    const value = await ownerKey();
    render(<Role.Select value={value} onChange={vi.fn()} {...props} />, { wrapper });
    const trigger = await screen.findByRole("button", { name: /Owner/ });
    await act(async () => {
      fireEvent.click(trigger);
    });
  };

  it("should show the icon of each built-in role in its dropdown", async () => {
    await open();
    for (const [name, icon] of Object.entries(BUILT_IN_ICONS)) {
      fireEvent.change(screen.getByPlaceholderText("Search roles..."), {
        target: { value: name },
      });
      // jsdom cannot place a floating dropdown, so it stays hidden.
      const option = await screen.findByRole("option", {
        name: new RegExp(`^${name}`),
        hidden: true,
      });
      expect(option.querySelector(`.lyra-icon--${icon}`)).not.toBeNull();
    }
  });

  it("should give a role without a built-in icon the generic role icon", () => {
    const { container } = render(Role.icon({ name: id.create() }));
    expect(container.querySelector(".lyra-icon--role")).not.toBeNull();
  });

  it("should keep its dropdown's width mark beside the caller's class", async () => {
    await open({ dialogProps: { className: "custom" } });
    const dialog = document.querySelector(".pluto-access-role__dialog");
    expect(dialog?.classList.contains("custom")).toBe(true);
  });
});
