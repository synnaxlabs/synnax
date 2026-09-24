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
import { id, sleep, TimeSpan } from "@synnaxlabs/x";
import { render, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";
import { Tooltip } from "@/schematic/Tooltip";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { Tooltip as Base } from "@/tooltip";
import { latestSample } from "@/vis/latestSample/aether";

const client = createTestClient();

const DELAY = TimeSpan.milliseconds(50);

const ANCHOR_RECT = {
  left: 100,
  top: 100,
  right: 200,
  bottom: 140,
  width: 100,
  height: 40,
  x: 100,
  y: 100,
  toJSON: () => ({}),
} as DOMRect;

const createAnchor = (): HTMLElement => {
  const el = document.createElement("div");
  el.getBoundingClientRect = () => ANCHOR_RECT;
  document.body.appendChild(el);
  return el;
};

interface Indexed {
  index: channel.Channel;
  data: channel.Channel;
}

const createIndex = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: id.create(),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const createIndexed = async (): Promise<Indexed> => {
  const index = await createIndex();
  const data = await client.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    index: index.key,
  });
  return { index, data };
};

const createChannel = async (): Promise<channel.Channel> =>
  (await createIndexed()).data;

const createVirtual = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    virtual: true,
  });

const createCalculated = async (): Promise<channel.Channel> => {
  const source = await createVirtual();
  return await client.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    virtual: true,
    expression: `return ${source.name} * 2`,
  });
};

const getTooltip = (): HTMLElement | null =>
  document.querySelector<HTMLElement>(".pluto-schematic-tooltip");

const findTooltip = async (): Promise<HTMLElement> => {
  await waitFor(() => expect(getTooltip()).not.toBeNull());
  return getTooltip() as HTMLElement;
};

const labelOf = (tooltip: HTMLElement, text: string): HTMLElement =>
  within(tooltip).getByText(text);

const valueOf = (label: HTMLElement): HTMLElement =>
  label.nextElementSibling as HTMLElement;

const dividersOf = (tooltip: HTMLElement): NodeListOf<Element> =>
  tooltip.querySelectorAll(".pluto-schematic-tooltip__divider");

describe("Schematic.Tooltip", () => {
  let Providers: FC<PropsWithChildren>;
  beforeAll(async () => {
    Providers = await createAsyncSynnaxWrapper({
      client,
      telemFactories: [],
      additionalRegistry: latestSample.REGISTRY,
    });
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Providers>
      <Base.Config delay={DELAY}>{children}</Base.Config>
    </Providers>
  );

  const renderTooltip = (config: Node.Config) =>
    render(<Tooltip anchor={createAnchor()} config={config} />, { wrapper: Wrapper });

  describe("timing", () => {
    it("should stay hidden until the delay passes", async () => {
      renderTooltip(Node.createConfig({ variant: "valve" }));
      expect(getTooltip()).toBeNull();
      await findTooltip();
    });

    it("should reopen instantly within the warm window", async () => {
      const config = Node.createConfig({ variant: "valve" });
      const { rerender } = renderTooltip(config);
      await findTooltip();
      rerender(<div />);
      expect(getTooltip()).toBeNull();
      rerender(<Tooltip anchor={createAnchor()} config={config} />);
      expect(getTooltip()).not.toBeNull();
    });
  });

  describe("channel rows", () => {
    it("should name the command and state channels under their role icons", async () => {
      const [command, state] = await Promise.all([createChannel(), createChannel()]);
      renderTooltip(
        Node.createConfig({
          variant: "valve",
          commandChannel: command.key,
          stateChannel: state.key,
        }),
      );
      const tooltip = await findTooltip();
      const commandLabel = labelOf(tooltip, command.name);
      expect(commandLabel.querySelector(".pluto-icon--edit")).not.toBeNull();
      const stateLabel = labelOf(tooltip, state.name);
      expect(stateLabel.querySelector(".pluto-icon--visible-filled")).not.toBeNull();
      expect(within(tooltip).getAllByText("f32")).toHaveLength(2);
    });

    it("should stay hidden until the channel names have loaded", async () => {
      const ch = await createChannel();
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
      const tooltip = await findTooltip();
      expect(within(tooltip).getByText(ch.name)).not.toBeNull();
      expect(within(tooltip).queryByText(String(ch.key))).toBeNull();
    });

    it("should show the tooltip when no channel is set", async () => {
      renderTooltip(Node.createConfig({ variant: "value" }));
      const tooltip = await findTooltip();
      expect(tooltip.querySelector(".pluto-icon--visible-filled")).toBeNull();
      expect(within(tooltip).getByText("Stale timeout")).not.toBeNull();
    });

    it("should show a plain channel without a kind icon", async () => {
      const ch = await createChannel();
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
      const tooltip = await findTooltip();
      const value = valueOf(labelOf(tooltip, ch.name));
      expect(value.textContent).toEqual("f32");
      expect(value.querySelector(".pluto-icon")).toBeNull();
    });

    it("should show an index channel as a channel row", async () => {
      const index = await createIndex();
      renderTooltip(Node.createConfig({ variant: "value", channel: index.key }));
      const tooltip = await findTooltip();
      const label = labelOf(tooltip, index.name);
      expect(label.querySelector(".pluto-icon--visible-filled")).not.toBeNull();
      expect(valueOf(label).textContent).toEqual("ts");
    });

    it("should mark a calculated channel with the calculation icon", async () => {
      const calc = await createCalculated();
      renderTooltip(Node.createConfig({ variant: "value", channel: calc.key }));
      const tooltip = await findTooltip();
      const value = valueOf(labelOf(tooltip, calc.name));
      expect(value.querySelector(".pluto-icon--calculation")).not.toBeNull();
    });

    it("should mark a virtual channel with the virtual icon", async () => {
      const virtual = await createVirtual();
      renderTooltip(Node.createConfig({ variant: "value", channel: virtual.key }));
      const tooltip = await findTooltip();
      const value = valueOf(labelOf(tooltip, virtual.name));
      expect(value.querySelector(".pluto-icon--virtual")).not.toBeNull();
    });
  });

  describe("last sample row", () => {
    it("should label the row under the time icon instead of the index name", async () => {
      const { index, data } = await createIndexed();
      renderTooltip(Node.createConfig({ variant: "value", channel: data.key }));
      const tooltip = await findTooltip();
      const label = labelOf(tooltip, "Last sample");
      expect(label.querySelector(".pluto-icon--time-outline")).not.toBeNull();
      expect(within(tooltip).queryByText(index.name)).toBeNull();
    });

    it("should close the tooltip in its own section", async () => {
      const { data } = await createIndexed();
      renderTooltip(Node.createConfig({ variant: "value", channel: data.key }));
      const tooltip = await findTooltip();
      const last = tooltip.lastElementChild as HTMLElement;
      expect(last.classList.contains("pluto-schematic-tooltip__last-write")).toBe(true);
      expect(last.contains(labelOf(tooltip, "Last sample"))).toBe(true);
      expect(last.previousElementSibling).toEqual(dividersOf(tooltip)[1]);
    });

    it("should leave the value blank until a sample is known", async () => {
      const { data } = await createIndexed();
      renderTooltip(Node.createConfig({ variant: "value", channel: data.key }));
      const tooltip = await findTooltip();
      await sleep.sleep(TimeSpan.milliseconds(250));
      expect(valueOf(labelOf(tooltip, "Last sample")).textContent).toEqual("");
    });

    it("should show one row for a valve with two indexed channels", async () => {
      const [command, state] = await Promise.all([createIndexed(), createIndexed()]);
      renderTooltip(
        Node.createConfig({
          variant: "valve",
          commandChannel: command.data.key,
          stateChannel: state.data.key,
        }),
      );
      const tooltip = await findTooltip();
      expect(within(tooltip).getAllByText("Last sample")).toHaveLength(1);
    });

    it("should fall back to the command channel", async () => {
      const command = await createIndexed();
      renderTooltip(
        Node.createConfig({ variant: "valve", commandChannel: command.data.key }),
      );
      const tooltip = await findTooltip();
      expect(within(tooltip).queryByText("Last sample")).not.toBeNull();
    });

    it("should show the row for a symbol bound to an index channel", async () => {
      const index = await createIndex();
      renderTooltip(Node.createConfig({ variant: "value", channel: index.key }));
      const tooltip = await findTooltip();
      expect(within(tooltip).queryByText("Last sample")).not.toBeNull();
    });

    it("should omit the row for a virtual channel", async () => {
      const virtual = await createVirtual();
      renderTooltip(Node.createConfig({ variant: "value", channel: virtual.key }));
      const tooltip = await findTooltip();
      expect(within(tooltip).queryByText("Last sample")).toBeNull();
    });

    it("should omit the row for a calculated channel", async () => {
      const calc = await createCalculated();
      renderTooltip(Node.createConfig({ variant: "value", channel: calc.key }));
      const tooltip = await findTooltip();
      expect(within(tooltip).queryByText("Last sample")).toBeNull();
    });
  });

  describe("field rows", () => {
    it("should label fields in sentence case with their units", async () => {
      renderTooltip(Node.createConfig({ variant: "valve", onClickDelay: 250 }));
      const tooltip = await findTooltip();
      expect(within(tooltip).getByText("On click delay")).not.toBeNull();
      expect(within(tooltip).getByText("250ms")).not.toBeNull();
      expect(within(tooltip).getByText("Stale timeout")).not.toBeNull();
      expect(within(tooltip).getByText("5s")).not.toBeNull();
    });

    it("should hide a zero click delay", async () => {
      renderTooltip(Node.createConfig({ variant: "valve" }));
      const tooltip = await findTooltip();
      expect(within(tooltip).queryByText("On click delay")).toBeNull();
    });

    it("should show a button's mode", async () => {
      renderTooltip(Node.createConfig({ variant: "button" }));
      const tooltip = await findTooltip();
      expect(within(tooltip).getByText("Mode")).not.toBeNull();
      expect(within(tooltip).getByText("fire")).not.toBeNull();
    });

    it("should show whether a solenoid valve is normally open", async () => {
      renderTooltip(
        Node.createConfig({ variant: "solenoid_valve", normallyOpen: true }),
      );
      const tooltip = await findTooltip();
      expect(within(tooltip).getByText("Normally open")).not.toBeNull();
      expect(within(tooltip).getByText("true")).not.toBeNull();
    });

    it.each<[Node.Variant, string[]]>([
      ["valve", ["Stale timeout"]],
      ["solenoid_valve", ["Normally open", "Stale timeout"]],
      ["button", ["Mode"]],
      ["value", ["Stale timeout"]],
      ["manual_valve", ["Clickable"]],
    ])("should show the %s rows in order", async (variant, labels) => {
      renderTooltip(Node.createConfig({ variant }));
      const tooltip = await findTooltip();
      const rendered = Array.from(
        tooltip.querySelectorAll(".pluto-schematic-tooltip__field"),
      ).map((el) => el.textContent);
      expect(rendered).toEqual(labels);
    });

    // A setpoint's only field is the click delay, which is hidden at zero.
    it.each<Node.Variant>(["cap", "tank", "circle", "group_box", "setpoint"])(
      "should render nothing for a %s, which has no rows",
      async (variant) => {
        renderTooltip(Node.createConfig({ variant }));
        await sleep.sleep(TimeSpan.milliseconds(DELAY.milliseconds * 3));
        expect(getTooltip()).toBeNull();
      },
    );
  });

  describe("dividers", () => {
    it("should divide channel rows from field rows", async () => {
      const virtual = await createVirtual();
      renderTooltip(Node.createConfig({ variant: "value", channel: virtual.key }));
      const tooltip = await findTooltip();
      expect(dividersOf(tooltip)).toHaveLength(1);
    });

    it("should divide the last sample row from the field rows", async () => {
      const { data } = await createIndexed();
      renderTooltip(Node.createConfig({ variant: "value", channel: data.key }));
      const tooltip = await findTooltip();
      expect(dividersOf(tooltip)).toHaveLength(2);
    });

    it("should omit the divider when there are no channel rows", async () => {
      renderTooltip(Node.createConfig({ variant: "value" }));
      const tooltip = await findTooltip();
      expect(dividersOf(tooltip)).toHaveLength(0);
    });
  });
});
