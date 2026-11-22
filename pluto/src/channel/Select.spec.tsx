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
import { fireEvent, render, type RenderResult, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it, onTestFinished, vi } from "vitest";

import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

const TRIGGER_CLASS = "pluto-channel__trigger";

describe("Channel select", () => {
  const Wrapper = createSynnaxWrapper({ client: null });

  it("should mark the single select's trigger so forms can widen it", () => {
    const c = render(<Channel.SelectSingle value={0} onChange={vi.fn()} />, {
      wrapper: Wrapper,
    });
    expect(c.container.querySelector(`.${TRIGGER_CLASS}`)).not.toBeNull();
  });

  it("should show the select's own icon when no channel is selected", () => {
    const c = render(
      <Channel.SelectSingle value={0} onChange={vi.fn()} icon={<Icon.Time />} />,
      { wrapper: Wrapper },
    );
    const trigger = c.container.querySelector(`.${TRIGGER_CLASS}`);
    expect(trigger?.querySelector(".lyra-icon--time")).not.toBeNull();
  });

  it("should mark the multiple select's trigger so forms can widen it", () => {
    const c = render(<Channel.SelectMultiple value={[]} onChange={vi.fn()} />, {
      wrapper: Wrapper,
    });
    expect(c.container.querySelector(`.${TRIGGER_CLASS}`)).not.toBeNull();
  });

  it("should keep the single select's trigger mark beside the caller's trigger props", () => {
    const c = render(
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

  it("should keep the multiple select's trigger mark beside the caller's trigger props", () => {
    const c = render(
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
      const c = render(
        <Channel.SelectSingle {...props} query={excluding(first.key)} />,
        { wrapper: Wrapper },
      );
      fireEvent.click(c.getByRole("button"));
      const search = await c.findByPlaceholderText(/search/i);
      fireEvent.change(search, { target: { value: term } });
      await c.findByText(second.name);
      expect(c.queryByText(first.name)).toBeNull();

      c.rerender(<Channel.SelectSingle {...props} query={excluding(second.key)} />);

      await c.findByText(first.name);
      expect(c.queryByText(second.name)).toBeNull();
      expect(c.getByText(third.name)).toBeTruthy();
      expect(c.getByPlaceholderText(/search/i)).toBe(search);
      expect(search).toHaveProperty("value", term);
    });
  });

  describe("calculatedHidden", () => {
    const LOOKUP = "/calculation/retrieve";
    const FAILED = "Failed to load channels";
    const client = createTestClient();
    let wrapper: FC<PropsWithChildren>;
    beforeAll(async () => {
      wrapper = await createAsyncSynnaxWrapper({ client });
    });

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

    const toggle = (c: RenderResult) => fireEvent.click(c.getAllByRole("button")[0]);

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
      const c = render(<Picker keys={[plain.key, output.key]} />, { wrapper });
      toggle(c);
      expect(await c.findByText(plain.name)).toBeTruthy();
      expect(c.queryByText(output.name)).toBeNull();
    });

    it("should send one lookup and list a channel once it answers", async () => {
      const plain = await createPlain();
      const { release, responded, sent } = holdResponses(client, LOOKUP);
      const c = render(<Picker keys={[plain.key]} />, { wrapper });
      toggle(c);
      await responded;
      expect(c.queryByText(plain.name)).toBeNull();
      release();
      expect(await c.findByText(plain.name)).toBeTruthy();
      expect(sent()).toBe(1);
    });

    it("should report a failure once while open, and say what to do", async () => {
      const plain = await createPlain();
      failRequests(client, LOOKUP, new Error("calculations unavailable"));
      const c = render(<Picker keys={[plain.key]} />, { wrapper });
      toggle(c);
      expect(await c.findByText(FAILED)).toBeTruthy();
      expect(c.getByText(/Open the picker again to retry/)).toBeTruthy();
      await waitFor(() => expect(reported.current).toEqual([FAILED]));
    });

    it("should show a failed lookup while the selection holds a channel without an answer", async () => {
      const selected = await createPlain();
      failLookups();
      const c = render(<Picker keys={[selected.key]} value={[selected.key]} />, {
        wrapper,
      });
      toggle(c);
      expect(await c.findByText(FAILED)).toBeTruthy();
      toggle(c);
      await waitFor(() => expect(c.queryByText(FAILED)).toBeNull());
      toggle(c);
      expect(await c.findByText(FAILED)).toBeTruthy();
    });

    it("should name a selected channel without an answer in its tag", async () => {
      const selected = await createPlain();
      failLookups();
      const c = render(<Picker keys={[selected.key]} value={[selected.key]} />, {
        wrapper,
      });
      expect(await c.findByText(selected.name)).toBeTruthy();
      expect(c.queryByText(String(selected.key))).toBeNull();
      toggle(c);
      expect(await c.findByText(FAILED)).toBeTruthy();
      expect(c.queryByText(String(selected.key))).toBeNull();
    });

    it("should not send lookups in a loop while the Core fails them", async () => {
      const plain = await createPlain();
      const { lookups } = failLookups(plain.key);
      const c = render(<Picker keys={[plain.key]} />, { wrapper });
      toggle(c);
      expect(await c.findByText(FAILED)).toBeTruthy();
      await new Promise((resolve) => setTimeout(resolve, 1000));
      expect(lookups()).toBe(1);
    });

    it("should keep showing a failed lookup when a channel joins the selection", async () => {
      const failing = await createPlain();
      const picked = await createPlain();
      await client.channels.retrieve(picked.key);
      await client.calculations.outputs.retrieve([picked.key]);
      failLookups();
      const c = render(<Picker keys={[failing.key]} />, { wrapper });
      toggle(c);
      expect(await c.findByText(FAILED)).toBeTruthy();
      c.rerender(<Picker keys={[failing.key]} value={[picked.key]} />);
      expect(await c.findByText(picked.name)).toBeTruthy();
      await new Promise((resolve) => setTimeout(resolve, 200));
      expect(c.queryByText(/No channels/)).toBeNull();
      expect(c.queryByText(FAILED)).not.toBeNull();
    });

    it("should keep showing a failed lookup when a selected channel is renamed", async () => {
      const term = id.create();
      const failing = await createPlain(`${term}_failing`);
      const picked = await createPlain();
      await client.channels.retrieve(picked.key);
      await client.calculations.outputs.retrieve([picked.key]);
      failLookups();
      const c = render(<Picker keys={[failing.key]} value={[picked.key]} />, {
        wrapper,
      });
      toggle(c);
      fireEvent.change(await c.findByPlaceholderText(/search/i), {
        target: { value: term },
      });
      expect(await c.findByText(FAILED)).toBeTruthy();
      const renamed = id.create();
      await client.channels.rename(picked.key, renamed);
      expect(await c.findByText(renamed)).toBeTruthy();
      await new Promise((resolve) => setTimeout(resolve, 200));
      expect(c.queryByText(/No channels/)).toBeNull();
      expect(c.queryByText(FAILED)).not.toBeNull();
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
      const c = render(<Picker keys={[failing.key]} />, { wrapper });
      toggle(c);
      expect(await c.findByText(FAILED)).toBeTruthy();
      // The open list takes in another channel, and its lookup answers.
      c.rerender(<Picker keys={[failing.key, later.key]} />);
      expect(await c.findByText(later.name)).toBeTruthy();
      expect(await c.findByText(failing.name)).toBeTruthy();
    });

    it("should look a failed channel up again when the picker opens again", async () => {
      const plain = await createPlain();
      const { restore } = failLookups();
      const c = render(<Picker keys={[plain.key]} />, { wrapper });
      toggle(c);
      expect(await c.findByText(FAILED)).toBeTruthy();
      restore();
      toggle(c);
      await new Promise((resolve) => setTimeout(resolve, 100));
      toggle(c);
      expect(await c.findByText(plain.name, {}, { timeout: 3000 })).toBeTruthy();
    });

    it("should report no failure that lands after the picker closes", async () => {
      const plain = await createPlain();
      const { release, sent } = holdFailures(client, LOOKUP, new Error("unavailable"));
      const c = render(<Picker keys={[plain.key]} />, { wrapper });
      toggle(c);
      await waitFor(() => expect(sent()).toBe(1));
      toggle(c);
      release();
      await new Promise((resolve) => setTimeout(resolve, 200));
      expect(reported.current).toEqual([]);
    });

    it("should report no failure that lands after the picker unmounts", async () => {
      const plain = await createPlain();
      const { release, sent } = holdFailures(client, LOOKUP, new Error("unavailable"));
      const c = render(<Picker keys={[plain.key]} />, { wrapper });
      toggle(c);
      await waitFor(() => expect(sent()).toBe(1));
      // The form that holds the open picker closes before the lookup fails.
      c.rerender(<Picker keys={[plain.key]} mounted={false} />);
      release();
      await new Promise((resolve) => setTimeout(resolve, 200));
      c.rerender(<Picker keys={[plain.key]} mounted={false} />);
      expect(reported.current).toEqual([]);
    });

    it("should show the list's empty state while it holds no records", async () => {
      const plain = await createPlain();
      failLookups();
      const term = id.create();
      const c = render(<Picker keys={[plain.key]} />, { wrapper });
      toggle(c);
      expect(await c.findByText(FAILED)).toBeTruthy();
      fireEvent.change(await c.findByPlaceholderText(/search/i), {
        target: { value: term },
      });
      expect(await c.findByText(new RegExp(term))).toBeTruthy();
      expect(c.queryByText(FAILED)).toBeNull();
    });
  });
});
