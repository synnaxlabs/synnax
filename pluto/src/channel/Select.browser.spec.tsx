// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type calculation, type channel, DataType } from "@synnaxlabs/client";
import {
  createTestClient,
  failRequests,
  holdFailures,
  holdResponses,
  rejectRequests,
} from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { id } from "@synnaxlabs/x";
import type { ReactElement } from "react";
import { describe, expect, it, onTestFinished, vi } from "vitest";
import { page, userEvent } from "vitest/browser";
import { render, type RenderResult } from "vitest-browser-react";

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

  describe("calculatedHidden", () => {
    const LOOKUP = "/calculation/retrieve";
    const FAILED = "Failed to load channels";
    const client = createTestClient();
    const Wrapper = createSynnaxWrapper({ client });

    const createPlain = async (name: string = id.create()) =>
      await client.channels.create({ name, dataType: DataType.FLOAT32, virtual: true });

    const reported: { current: string[] } = { current: [] };
    const Reader = (): null => {
      reported.current = Status.useNotifications().statuses.map(
        ({ message }) => message,
      );
      return null;
    };

    interface PickerProps {
      keys: channel.Key[];
      value?: channel.Key[];
      mounted?: boolean;
    }

    const Picker = ({
      keys,
      value = [],
      mounted = true,
    }: PickerProps): ReactElement => (
      <>
        <Reader />
        {mounted && (
          <Channel.SelectMultiple
            value={value}
            onChange={vi.fn()}
            query={{ where: { key: { in: keys } } }}
            calculatedHidden
            virtual={false}
          />
        )}
      </>
    );

    const toggle = async (c: RenderResult) =>
      await c.getByRole("button").first().click();
    const text = (t: string | RegExp) =>
      page.getByText(t, { exact: typeof t === "string" });
    const shown = async (t: string | RegExp, timeout?: number) =>
      await expect.element(text(t), { timeout }).toBeInTheDocument();
    const absent = (t: string | RegExp) => text(t).query() == null;
    const sleep = async (ms: number) =>
      await new Promise((resolve) => setTimeout(resolve, ms));

    // Fails each lookup, and counts the lookups of counted. A spec before this one can
    // still look its own channels up.
    const failLookups = (counted?: channel.Key) => {
      let lookups = 0;
      const spy = rejectRequests(
        client,
        (target, req) => {
          if (target !== LOOKUP) return false;
          const { where } = req as calculation.RetrieveRequest;
          if (counted != null && where?.output?.in?.includes(counted) === true)
            lookups++;
          return true;
        },
        new Error("calculations unavailable"),
      );
      onTestFinished(() => spy.mockRestore());
      return { lookups: () => lookups, restore: () => spy.mockRestore() };
    };

    it("should list only the channels that no calculation writes", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      const output = await client.channels.retrieve(calc.output);
      const plain = await createPlain();
      const c = await render(<Picker keys={[plain.key, output.key]} />, {
        wrapper: Wrapper,
      });
      await toggle(c);
      await shown(plain.name);
      expect(absent(output.name)).toBe(true);
    });

    it("should send one lookup and list a channel once it answers", async () => {
      const plain = await createPlain();
      const { release, responded, sent } = holdResponses(client, LOOKUP);
      const c = await render(<Picker keys={[plain.key]} />, { wrapper: Wrapper });
      await toggle(c);
      await responded;
      expect(absent(plain.name)).toBe(true);
      release();
      await shown(plain.name);
      expect(sent()).toBe(1);
    });

    it("should report a failure once while open, and say what to do", async () => {
      const plain = await createPlain();
      failRequests(client, LOOKUP, new Error("calculations unavailable"));
      const c = await render(<Picker keys={[plain.key]} />, { wrapper: Wrapper });
      await toggle(c);
      await shown(FAILED);
      await shown(/Open the picker again to retry/);
      await expect.poll(() => reported.current).toEqual([FAILED]);
    });

    it("should show a failed lookup while the selection holds a channel without an answer", async () => {
      const selected = await createPlain();
      failLookups();
      const c = await render(<Picker keys={[selected.key]} value={[selected.key]} />, {
        wrapper: Wrapper,
      });
      await toggle(c);
      await shown(FAILED);
      await toggle(c);
      await expect.element(text(FAILED)).not.toBeInTheDocument();
      await toggle(c);
      await shown(FAILED);
    });

    it("should name a selected channel without an answer in its tag", async () => {
      const selected = await createPlain();
      failLookups();
      const c = await render(<Picker keys={[selected.key]} value={[selected.key]} />, {
        wrapper: Wrapper,
      });
      await shown(selected.name);
      expect(absent(String(selected.key))).toBe(true);
      await toggle(c);
      await shown(FAILED);
      expect(absent(String(selected.key))).toBe(true);
    });

    it("should not send lookups in a loop while the Core fails them", async () => {
      const plain = await createPlain();
      const { lookups } = failLookups(plain.key);
      const c = await render(<Picker keys={[plain.key]} />, { wrapper: Wrapper });
      await toggle(c);
      await shown(FAILED);
      await sleep(1000);
      expect(lookups()).toBe(1);
    });

    it("should keep showing a failed lookup when a channel joins the selection", async () => {
      const failing = await createPlain();
      const picked = await createPlain();
      await client.channels.retrieve(picked.key);
      await client.calculations.outputs.retrieve([picked.key]);
      failLookups();
      const c = await render(<Picker keys={[failing.key]} />, { wrapper: Wrapper });
      await toggle(c);
      await shown(FAILED);
      await c.rerender(<Picker keys={[failing.key]} value={[picked.key]} />);
      await shown(picked.name);
      await sleep(200);
      expect(absent(/No channels/)).toBe(true);
      expect(absent(FAILED)).toBe(false);
    });

    it("should keep showing a failed lookup when a selected channel is renamed", async () => {
      const term = id.create();
      const failing = await createPlain(`${term}_failing`);
      const picked = await createPlain();
      await client.channels.retrieve(picked.key);
      await client.calculations.outputs.retrieve([picked.key]);
      failLookups();
      const c = await render(<Picker keys={[failing.key]} value={[picked.key]} />, {
        wrapper: Wrapper,
      });
      await toggle(c);
      await userEvent.fill(page.getByPlaceholder(/search/i), term);
      await shown(FAILED);
      const renamed = id.create();
      await client.channels.rename(picked.key, renamed);
      await shown(renamed);
      await sleep(200);
      expect(absent(/No channels/)).toBe(true);
      expect(absent(FAILED)).toBe(false);
    });

    it("should look a failed channel up again once a later lookup answers", async () => {
      const failing = await createPlain();
      const later = await createPlain();
      let failures = 1;
      const spy = rejectRequests(
        client,
        (target, req) => {
          if (target !== LOOKUP || failures === 0) return false;
          const { where } = req as calculation.RetrieveRequest;
          if (where?.output?.in?.includes(failing.key) !== true) return false;
          failures--;
          return true;
        },
        new Error("calculations unavailable"),
      );
      onTestFinished(() => spy.mockRestore());
      const c = await render(<Picker keys={[failing.key]} />, { wrapper: Wrapper });
      await toggle(c);
      await shown(FAILED);
      // The open list takes in another channel, and its lookup answers.
      await c.rerender(<Picker keys={[failing.key, later.key]} />);
      await shown(later.name);
      await shown(failing.name);
    });

    it("should look a failed channel up again when the picker opens again", async () => {
      const plain = await createPlain();
      const { restore } = failLookups();
      const c = await render(<Picker keys={[plain.key]} />, { wrapper: Wrapper });
      await toggle(c);
      await shown(FAILED);
      restore();
      await toggle(c);
      await sleep(100);
      await toggle(c);
      await shown(plain.name, 3000);
    });

    it("should report no failure that lands after the picker closes", async () => {
      const plain = await createPlain();
      const { release, sent } = holdFailures(client, LOOKUP, new Error("unavailable"));
      const c = await render(<Picker keys={[plain.key]} />, { wrapper: Wrapper });
      await toggle(c);
      await expect.poll(sent).toBe(1);
      await toggle(c);
      release();
      await sleep(200);
      expect(reported.current).toEqual([]);
    });

    it("should report no failure that lands after the picker unmounts", async () => {
      const plain = await createPlain();
      const { release, sent } = holdFailures(client, LOOKUP, new Error("unavailable"));
      const c = await render(<Picker keys={[plain.key]} />, { wrapper: Wrapper });
      await toggle(c);
      await expect.poll(sent).toBe(1);
      // The form that holds the open picker closes before the lookup fails.
      await c.rerender(<Picker keys={[plain.key]} mounted={false} />);
      release();
      await sleep(200);
      await c.rerender(<Picker keys={[plain.key]} mounted={false} />);
      expect(reported.current).toEqual([]);
    });

    it("should show the list's empty state while it holds no records", async () => {
      const plain = await createPlain();
      failLookups();
      const term = id.create();
      const c = await render(<Picker keys={[plain.key]} />, { wrapper: Wrapper });
      await toggle(c);
      await shown(FAILED);
      await userEvent.fill(page.getByPlaceholder(/search/i), term);
      await shown(new RegExp(term));
      expect(absent(FAILED)).toBe(true);
    });
  });
});
