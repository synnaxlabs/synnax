// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { id } from "@synnaxlabs/x";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";
import { render } from "vitest-browser-react";

import { Channel } from "@/channel";
import { createSynnaxWrapper } from "@/testutil";

const TRIGGER_CLASS = "pluto-channel__trigger";

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

  describe("drag", () => {
    const client = createTestClient();
    const Wrapper = createSynnaxWrapper({ client });

    const Dragged = (): ReactElement => (
      <output>{JSON.stringify(Haul.useDraggingState().items)}</output>
    );

    const createTextChannel = async (): Promise<channel.Channel> => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      return await client.channels.create({
        name: id.create(),
        dataType: DataType.STRING,
        index: index.key,
      });
    };

    const dragStart = (el: Element): void => {
      el.dispatchEvent(
        new DragEvent("dragstart", { bubbles: true, dataTransfer: new DataTransfer() }),
      );
    };

    const expectDragged = async (text: channel.Channel): Promise<void> => {
      await expect
        .element(page.getByRole("status"))
        .toHaveTextContent(
          JSON.stringify([
            { type: Channel.HAUL_TYPE, key: text.key, data: { plottable: false } },
          ]),
        );
    };

    it("should record whether a line plot can draw the channel a single select drags", async () => {
      const text = await createTextChannel();
      const c = await render(
        <Haul.Provider>
          <Channel.SelectSingle value={text.key} onChange={vi.fn()} />
          <Dragged />
        </Haul.Provider>,
        { wrapper: Wrapper },
      );
      const trigger = c.container.querySelector(`.${TRIGGER_CLASS}`);
      if (trigger == null) throw new Error("the trigger did not render");
      await expect.element(page.getByText(text.name)).toBeInTheDocument();
      dragStart(trigger);
      await expectDragged(text);
    });

    it("should record whether a line plot can draw the channel a multiple select drags", async () => {
      const text = await createTextChannel();
      await render(
        <Haul.Provider>
          <Channel.SelectMultiple value={[text.key]} onChange={vi.fn()} />
          <Dragged />
        </Haul.Provider>,
        { wrapper: Wrapper },
      );
      const tag = page.getByText(text.name);
      await expect.element(tag).toBeInTheDocument();
      dragStart(tag.element());
      await expectDragged(text);
    });
  });
});
