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
import { Icon } from "@synnaxlabs/lyra/icon";
import { id } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";
import { render } from "vitest-browser-react";

import { Channel } from "@/channel";
import { createSynnaxWrapper } from "@/testutil/Synnax";

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
});
