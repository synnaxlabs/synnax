// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-react";

import { Group } from "@/group";
import { createSynnaxWrapper } from "@/testutil";

const PARENTS_CLASS = "pluto-group__parents";

describe("Group select", () => {
  const client = createTestClient();
  const Wrapper = createSynnaxWrapper({ client });

  const createGroup = async (parent?: group.Group): Promise<group.Group> =>
    await client.groups.create(
      parent == null
        ? { name: id.create(), scope: "channel" }
        : { name: id.create(), parent: group.resourceID(parent.key) },
    );

  const shownParents = (): string[] =>
    Array.from(document.querySelectorAll(`.${PARENTS_CLASS}`), (e) => e.textContent)
      .filter((text): text is string => text != null)
      .sort();

  it("should show the groups above each listed group", async () => {
    const outer = await createGroup();
    const middle = await createGroup(outer);
    const inner = await createGroup(middle);
    const query = { where: { key: { in: [outer.key, middle.key, inner.key] } } };
    const c = await render(
      <Group.SelectSingle value={undefined} onChange={vi.fn()} query={query} />,
      { wrapper: Wrapper },
    );
    await c.getByRole("button").click();
    await expect
      .poll(shownParents)
      .toEqual([outer.name, `${outer.name}.${middle.name}`].sort());
  });

  it("should not widen a trigger that sizes to its content for the groups above", async () => {
    let parent = await createGroup();
    for (let i = 0; i < 3; i++) parent = await createGroup(parent);
    const c = await render(
      <div style={{ display: "inline-flex" }}>
        <Group.SelectSingle value={parent.key} onChange={vi.fn()} />
      </div>,
      { wrapper: Wrapper },
    );
    await expect.poll(() => c.container.textContent ?? "").toContain(parent.name);
    const end = c.container.querySelector<HTMLElement>(
      ".lyra-select-single-trigger__end",
    );
    expect(end?.clientWidth).toEqual(0);
  });

  it("should show the groups above the selected group in the trigger", async () => {
    const outer = await createGroup();
    const inner = await createGroup(outer);
    const c = await render(
      <Group.SelectSingle
        value={inner.key}
        onChange={vi.fn()}
        query={{ where: { key: { in: [outer.key] } } }}
      />,
      { wrapper: Wrapper },
    );
    const triggerText = () => c.container.textContent ?? "";
    await expect.poll(triggerText).toContain(inner.name);
    await expect.poll(triggerText).toContain(outer.name);
  });
});
