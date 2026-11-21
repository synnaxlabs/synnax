// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { access } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import type { FC, PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";
import { render } from "vitest-browser-react";

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
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const open = async (props: Partial<Role.SelectProps> = {}): Promise<void> => {
    const value = await ownerKey();
    await render(<Role.Select value={value} onChange={vi.fn()} {...props} />, {
      wrapper,
    });
    await page.getByRole("button", { name: /Owner/ }).click();
  };

  it("should show the icon of each built-in role in its dropdown", async () => {
    await open();
    for (const [name, icon] of Object.entries(BUILT_IN_ICONS)) {
      await userEvent.fill(page.getByPlaceholder("Search roles..."), name);
      const option = page.getByRole("option", { name: new RegExp(`^${name}`) });
      await expect.element(option).toBeInTheDocument();
      expect(option.element().querySelector(`.lyra-icon--${icon}`)).not.toBeNull();
    }
  });

  it("should give a role without a built-in icon the generic role icon", async () => {
    const { container } = await render(Role.icon({ name: id.create() }));
    expect(container.querySelector(".lyra-icon--role")).not.toBeNull();
  });

  it("should keep its dropdown's width mark beside the caller's class", async () => {
    await open({ dialogProps: { className: "custom" } });
    await expect
      .poll(() =>
        document
          .querySelector(".pluto-access-role__dialog")
          ?.classList.contains("custom"),
      )
      .toBe(true);
  });
});
