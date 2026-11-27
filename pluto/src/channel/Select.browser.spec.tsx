// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, group } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/lyra/icon";
import { id } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";
import { render } from "vitest-browser-react";

import { Channel } from "@/channel";
import { createSynnaxWrapper } from "@/testutil";

const TRIGGER_CLASS = "pluto-channel__trigger";
const FOLDER_CLASS = "pluto-channel__folder";

describe("Channel select", () => {
  const Wrapper = createSynnaxWrapper({ client: null });

  it("should mark the single select's trigger so forms can widen it", async () => {
    const c = await render(<Channel.SelectSingle value={0} onChange={vi.fn()} />, {
      wrapper: Wrapper,
    });
    expect(c.container.querySelector(`.${TRIGGER_CLASS}`)).not.toBeNull();
  });

  it("should show the select's own icon when no channel is selected", async () => {
    const c = await render(
      <Channel.SelectSingle value={0} onChange={vi.fn()} icon={<Icon.Time />} />,
      { wrapper: Wrapper },
    );
    const trigger = c.container.querySelector(`.${TRIGGER_CLASS}`);
    expect(trigger?.querySelector(".lyra-icon--time")).not.toBeNull();
  });

  it("should mark the multiple select's trigger so forms can widen it", async () => {
    const c = await render(<Channel.SelectMultiple value={[]} onChange={vi.fn()} />, {
      wrapper: Wrapper,
    });
    expect(c.container.querySelector(`.${TRIGGER_CLASS}`)).not.toBeNull();
  });

  it("should keep the single select's trigger mark beside the caller's trigger props", async () => {
    const c = await render(
      <Channel.SelectSingle
        value={0}
        onChange={vi.fn()}
        triggerProps={{ variant: "shadow", className: "custom" }}
      />,
      { wrapper: Wrapper },
    );
    const trigger = c.container.querySelector(`.${TRIGGER_CLASS}`);
    expect(trigger?.classList.contains("custom")).toBe(true);
    expect(trigger?.classList.contains("lyra-btn--shadow")).toBe(true);
  });

  it("should keep the multiple select's trigger mark beside the caller's trigger props", async () => {
    const c = await render(
      <Channel.SelectMultiple
        value={[]}
        onChange={vi.fn()}
        triggerProps={{ className: "custom" }}
      />,
      { wrapper: Wrapper },
    );
    const trigger = c.container.querySelector(`.${TRIGGER_CLASS}`);
    expect(trigger?.classList.contains("custom")).toBe(true);
  });

  describe("paths", () => {
    const client = createTestClient();
    const Wrapper = createSynnaxWrapper({ client });

    const createFolder = async (parent?: group.Group): Promise<group.Group> =>
      await client.groups.create(
        parent == null
          ? { name: id.create(), scope: "channel" }
          : { name: id.create(), parent: group.resourceID(parent.key) },
      );

    const createChannel = async (folder?: group.Group): Promise<channel.Channel> =>
      await client.channels.create({
        name: `path_${id.create()}`,
        dataType: DataType.FLOAT32,
        virtual: true,
        group: folder?.key,
      });

    const expectFolder = async (text: string): Promise<void> => {
      const folder = page.getByText(text, { exact: true });
      await expect.element(folder).toBeVisible();
      await expect.element(folder).toHaveClass(FOLDER_CLASS);
    };

    it("should show the folders of each listed channel's path", async () => {
      const outer = await createFolder();
      const inner = await createFolder(outer);
      const other = await createFolder();
      const [nested, sibling, top] = await Promise.all([
        createChannel(inner),
        createChannel(other),
        createChannel(),
      ]);
      const query = { where: { key: { in: [nested.key, sibling.key, top.key] } } };
      const c = await render(
        <Channel.SelectSingle value={0} onChange={vi.fn()} query={query} />,
        { wrapper: Wrapper },
      );
      await c.getByRole("button").click();
      await expect.element(page.getByText(top.name, { exact: true })).toBeVisible();
      await expectFolder(`${outer.name}.${inner.name}`);
      await expectFolder(other.name);
      expect(document.querySelectorAll(`.${FOLDER_CLASS}`)).toHaveLength(2);
    });

    it("should show the folders of the selected channel in the trigger", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const c = await render(
        <Channel.SelectSingle value={ch.key} onChange={vi.fn()} />,
        { wrapper: Wrapper },
      );
      const triggerText = () =>
        c.container.querySelector(`.${TRIGGER_CLASS}`)?.textContent ?? "";
      await expect.poll(triggerText).toContain(ch.name);
      expect(triggerText()).toContain(folder.name);
    });

    it("should fade the folders before the name in a narrow trigger", async () => {
      let folder = await client.groups.create({
        name: `Instrumentation and sensors ${id.create()}`,
        scope: "channel",
      });
      for (let i = 0; i < 3; i++)
        folder = await client.groups.create({
          name: `Pressure transducers bank ${i} ${id.create()}`,
          parent: group.resourceID(folder.key),
        });
      const ch = await createChannel(folder);
      const c = await render(
        <div style={{ width: "220px" }}>
          <Channel.SelectSingle value={ch.key} onChange={vi.fn()} />
        </div>,
        { wrapper: Wrapper },
      );
      const trigger = () => c.container.querySelector<HTMLElement>(`.${TRIGGER_CLASS}`);
      await expect
        .poll(() => trigger()?.querySelector(`.${FOLDER_CLASS}`))
        .not.toBeNull();
      const label = trigger()?.querySelector<HTMLElement>(".lyra-select__label");
      expect(label?.textContent).toEqual(ch.name);
      expect(label?.clientWidth).toBeGreaterThan(60);
      const el = trigger();
      expect(el?.scrollWidth).toBeLessThanOrEqual(el?.clientWidth ?? 0);
    });

    it("should not widen a trigger that sizes to its content for the folders", async () => {
      const folder = await client.groups.create({
        name: `Instrumentation and sensors ${id.create()}`,
        scope: "channel",
      });
      const ch = await createChannel(folder);
      const c = await render(
        <div style={{ display: "inline-flex" }}>
          <Channel.SelectSingle value={ch.key} onChange={vi.fn()} />
        </div>,
        { wrapper: Wrapper },
      );
      const trigger = () => c.container.querySelector<HTMLElement>(`.${TRIGGER_CLASS}`);
      await expect.poll(() => trigger()?.textContent ?? "").toContain(ch.name);
      const end = trigger()?.querySelector<HTMLElement>(
        ".lyra-select-single-trigger__end",
      );
      expect(end?.clientWidth).toEqual(0);
    });

    it("should show the folders of each channel in a multiple select", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const c = await render(
        <Channel.SelectMultiple
          value={[]}
          onChange={vi.fn()}
          query={{ where: { key: { in: [ch.key] } } }}
        />,
        { wrapper: Wrapper },
      );
      await c.getByRole("button").click();
      await expectFolder(folder.name);
    });
  });

  describe("query changes", () => {
    const client = createTestClient();
    const Wrapper = createSynnaxWrapper({ client });

    it("should keep an open dialog and its search term and show the new results", async () => {
      const term = id.create();
      const [first, second, third] = await client.channels.create(
        ["first", "second", "third"].map((suffix) => ({
          name: `${term}_${suffix}`,
          dataType: DataType.FLOAT32,
          virtual: true,
        })),
      );
      const excluding = (key: channel.Key): Channel.ListQuery => ({
        where: { not: { key: { in: [key] } } },
      });
      const props = { value: 0, onChange: vi.fn(), virtual: false };
      const c = await render(
        <Channel.SelectSingle {...props} query={excluding(first.key)} />,
        { wrapper: Wrapper },
      );
      await c.getByRole("button").click();
      const search = page.getByPlaceholder(/search/i);
      await userEvent.fill(search, term);
      const searchElement = search.element();
      const text = (name: string) => page.getByText(name, { exact: true });
      await expect.element(text(second.name)).toBeInTheDocument();
      await expect.element(text(first.name)).not.toBeInTheDocument();

      await c.rerender(
        <Channel.SelectSingle {...props} query={excluding(second.key)} />,
      );

      await expect.element(text(first.name)).toBeInTheDocument();
      await expect.element(text(second.name)).not.toBeInTheDocument();
      await expect.element(text(third.name)).toBeInTheDocument();
      expect(search.element()).toBe(searchElement);
      await expect.element(search).toHaveValue(term);
    });
  });
});
