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
import { id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { AliasInput } from "@/channel/AliasInput";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createChannel = async (): Promise<channel.Channel> => {
  const index = await client.channels.create({
    name: id.create(),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  return await client.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    index: index.key,
  });
};

const createRange = () =>
  client.ranges.create({
    name: id.create(),
    timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
  });

const DISABLED_CLASS = "pluto--disabled";

const getButtons = (container: HTMLElement): HTMLButtonElement[] =>
  Array.from(container.querySelectorAll("button"));

const getSetButton = (container: HTMLElement): HTMLButtonElement =>
  getButtons(container).at(-1) as HTMLButtonElement;

describe("channel/AliasInput", () => {
  let Wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    Wrapper = await createAsyncSynnaxWrapper({ client });
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  const renderInput = (props: Record<string, unknown> = {}) =>
    render(<AliasInput channel={1} value="" onChange={vi.fn()} {...props} />, {
      wrapper: Wrapper,
    });

  describe("set alias button", () => {
    it("should persist the alias on the range when clicked", async () => {
      const ch = await createChannel();
      const range = await createRange();
      const alias = id.create();
      const { container } = renderInput({
        channel: ch.key,
        range: range.key,
        value: alias,
      });
      fireEvent.click(getSetButton(container));
      await waitFor(async () =>
        expect(await client.ranges.retrieveAlias(range.key, ch.key)).toEqual(alias),
      );
    });

    it("should be disabled when the channel is zero", () => {
      const { container } = renderInput({ channel: 0, value: "vibes" });
      expect(getSetButton(container).classList).toContain(DISABLED_CLASS);
    });

    it("should become disabled once the stored alias matches the value", async () => {
      const ch = await createChannel();
      const range = await createRange();
      const alias = id.create();
      await client.ranges.setAlias(range.key, ch.key, alias);
      const { container } = renderInput({
        channel: ch.key,
        range: range.key,
        value: alias,
      });
      await waitFor(() =>
        expect(getSetButton(container).classList).toContain(DISABLED_CLASS),
      );
    });
  });

  describe("reset button", () => {
    it("should render and invoke onReset when an override is present", () => {
      const onReset = vi.fn();
      const { container } = renderInput({ value: "vibes", isDefault: false, onReset });
      const buttons = getButtons(container);
      expect(buttons).toHaveLength(2);
      fireEvent.click(buttons[0]);
      expect(onReset).toHaveBeenCalledTimes(1);
    });

    it("should not render when the value is the derived default", () => {
      const onReset = vi.fn();
      const { container } = renderInput({ value: "vibes", isDefault: true, onReset });
      expect(getButtons(container)).toHaveLength(1);
    });

    it("should not render when no onReset is provided", () => {
      const { container } = renderInput({ value: "vibes", isDefault: false });
      expect(getButtons(container)).toHaveLength(1);
    });
  });
});
