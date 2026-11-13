// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, connection, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Tooltip as Base } from "@synnaxlabs/lyra/tooltip";
import { id, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { render, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";
import { Tooltip } from "@/schematic/Tooltip";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
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

const createIndex = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: id.create(),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const createChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    index: (await createIndex()).key,
  });

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

const STREAM_VALUE_TYPE = telem.streamChannelValue({ channel: 0 }).type;

// The harness has no worker client, so this stands in for the remote factory and
// reports the sample times a spec sets per channel.
class StampFactory implements telem.Factory {
  type = "remote";
  private readonly stamps = new Map<channel.Key, TimeStamp>();
  private readonly sources = new Map<channel.Key, telemTest.TestSource<number>[]>();

  stamp(key: channel.Key, at: TimeStamp): void {
    this.stamps.set(key, at);
    this.sources.get(key)?.forEach((s) => {
      s.time = at;
      s.setValue(s.value().data + 1);
    });
  }

  create(spec: telem.Spec): telem.Telem | null {
    if (spec.type !== STREAM_VALUE_TYPE) return null;
    const key = spec.props.channel as channel.Key;
    const source = telemTest.source<number>(1);
    source.time = this.stamps.get(key);
    this.sources.set(key, [...(this.sources.get(key) ?? []), source]);
    return source;
  }
}

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
  let stamps: StampFactory;
  beforeAll(async () => {
    stamps = new StampFactory();
    Providers = await createAsyncSynnaxWrapper({
      client,
      telemFactories: [stamps],
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
      expect(commandLabel.querySelector(".lyra-icon--control")).not.toBeNull();
      const stateLabel = labelOf(tooltip, state.name);
      expect(stateLabel.querySelector(".lyra-icon--channel")).not.toBeNull();
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
      expect(tooltip.querySelector(".lyra-icon--channel")).toBeNull();
      expect(within(tooltip).getByText("Stale timeout")).not.toBeNull();
    });

    it("should show a plain channel without a kind icon", async () => {
      const ch = await createChannel();
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
      const tooltip = await findTooltip();
      const value = valueOf(labelOf(tooltip, ch.name));
      expect(value.textContent).toEqual("f32");
      expect(value.querySelector(".lyra-icon")).toBeNull();
    });

    it("should mark an index channel with the time icon", async () => {
      const index = await createIndex();
      renderTooltip(Node.createConfig({ variant: "value", channel: index.key }));
      const tooltip = await findTooltip();
      const value = valueOf(labelOf(tooltip, index.name));
      expect(value.querySelector(".lyra-icon--time")).not.toBeNull();
    });

    it("should mark a calculated channel with the calculation icon", async () => {
      const calc = await createCalculated();
      renderTooltip(Node.createConfig({ variant: "value", channel: calc.key }));
      const tooltip = await findTooltip();
      const value = valueOf(labelOf(tooltip, calc.name));
      expect(value.querySelector(".lyra-icon--calculation")).not.toBeNull();
    });

    it("should mark a virtual channel with the virtual icon", async () => {
      const virtual = await createVirtual();
      renderTooltip(Node.createConfig({ variant: "value", channel: virtual.key }));
      const tooltip = await findTooltip();
      const value = valueOf(labelOf(tooltip, virtual.name));
      expect(value.querySelector(".lyra-icon--virtual")).not.toBeNull();
    });
  });

  describe("field rows", () => {
    it("should label fields in sentence case with their units", async () => {
      renderTooltip(
        Node.createConfig({
          variant: "valve",
          onClickDelay: TimeSpan.milliseconds(250).nanoseconds,
        }),
      );
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
      ["scale", ["Stale timeout"]],
      ["tank", ["Stale timeout"]],
      ["setpoint", ["Stale timeout"]],
      ["input", ["Stale timeout"]],
      ["select", ["Stale timeout"]],
    ])("should show the %s rows in order", async (variant, labels) => {
      renderTooltip(Node.createConfig({ variant }));
      const tooltip = await findTooltip();
      const rendered = within(tooltip)
        .getAllByRole("term")
        .map((el) => el.textContent);
      expect(rendered).toEqual(labels);
    });

    it.each<Node.Variant>(["cap", "circle", "group_box"])(
      "should render nothing for a %s, which has no rows",
      async (variant) => {
        renderTooltip(Node.createConfig({ variant }));
        await sleep.sleep(TimeSpan.milliseconds(DELAY.milliseconds * 3));
        expect(getTooltip()).toBeNull();
      },
    );
  });

  describe("sample age row", () => {
    const LABEL = "Sample age";
    const findRowValue = async (tooltip: HTMLElement): Promise<string | null> => {
      await waitFor(() =>
        expect(valueOf(labelOf(tooltip, LABEL)).textContent).not.toEqual(""),
      );
      return valueOf(labelOf(tooltip, LABEL)).textContent;
    };

    it("should label the row under the time icon", async () => {
      const ch = await createChannel();
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
      const tooltip = await findTooltip();
      const label = labelOf(tooltip, LABEL);
      expect(label.querySelector(".lyra-icon--time-outline")).not.toBeNull();
    });

    it("should close the tooltip in its own section", async () => {
      const ch = await createChannel();
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
      const tooltip = await findTooltip();
      const last = tooltip.lastElementChild as HTMLElement;
      expect(last.contains(labelOf(tooltip, LABEL))).toBe(true);
      expect(last.previousElementSibling).toEqual(dividersOf(tooltip)[1]);
    });

    it("should leave the value blank until a sample is known", async () => {
      const ch = await createChannel();
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
      const tooltip = await findTooltip();
      await sleep.sleep(TimeSpan.milliseconds(250));
      expect(valueOf(labelOf(tooltip, LABEL)).textContent).toEqual("");
    });

    it("should show the time since the newest sample", async () => {
      const ch = await createChannel();
      stamps.stamp(ch.key, TimeStamp.now().sub(TimeSpan.hours(2)));
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
      const tooltip = await findTooltip();
      expect(await findRowValue(tooltip)).toEqual("2h");
    });

    it("should remove the clock skew from the age", async () => {
      // Ten minutes of skew would otherwise read as "2h 10min".
      const clockSkew = TimeSpan.minutes(10);
      const connectionStatus: connection.Status = {
        ...connection.DEFAULT_STATUS,
        variant: "success",
        details: { ...connection.DEFAULT_STATUS.details, clockSkew },
      };
      const Skewed = await createAsyncSynnaxWrapper({
        client,
        telemFactories: [stamps],
        additionalRegistry: latestSample.REGISTRY,
        connectionStatus,
      });
      const ch = await createChannel();
      stamps.stamp(ch.key, TimeStamp.now().sub(clockSkew).sub(TimeSpan.hours(2)));
      render(
        <Tooltip
          anchor={createAnchor()}
          config={Node.createConfig({ variant: "value", channel: ch.key })}
        />,
        {
          wrapper: ({ children }) => (
            <Skewed>
              <Base.Config delay={DELAY}>{children}</Base.Config>
            </Skewed>
          ),
        },
      );
      const tooltip = await findTooltip();
      expect(await findRowValue(tooltip)).toEqual("2h");
    });

    it("should follow a sample that arrives while open", async () => {
      const ch = await createChannel();
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
      const tooltip = await findTooltip();
      // The row's clock ticks once a second, so the stamp sits past the hour mark.
      stamps.stamp(ch.key, TimeStamp.now().sub(TimeSpan.hours(1).add(TimeSpan.SECOND)));
      expect(await findRowValue(tooltip)).toEqual("1h");
    });

    it("should show one row for a valve with two channels", async () => {
      const [command, state] = await Promise.all([createChannel(), createChannel()]);
      renderTooltip(
        Node.createConfig({
          variant: "valve",
          commandChannel: command.key,
          stateChannel: state.key,
        }),
      );
      const tooltip = await findTooltip();
      expect(within(tooltip).getAllByText(LABEL)).toHaveLength(1);
    });

    it("should follow the state channel before the command channel", async () => {
      const [command, state] = await Promise.all([createChannel(), createChannel()]);
      stamps.stamp(command.key, TimeStamp.now().sub(TimeSpan.hours(1)));
      stamps.stamp(state.key, TimeStamp.now().sub(TimeSpan.hours(2)));
      renderTooltip(
        Node.createConfig({
          variant: "valve",
          commandChannel: command.key,
          stateChannel: state.key,
        }),
      );
      const tooltip = await findTooltip();
      expect(await findRowValue(tooltip)).toEqual("2h");
    });

    it("should fall back to the command channel", async () => {
      const command = await createChannel();
      renderTooltip(
        Node.createConfig({ variant: "valve", commandChannel: command.key }),
      );
      const tooltip = await findTooltip();
      expect(within(tooltip).queryByText(LABEL)).not.toBeNull();
    });

    it.each<[string, () => Promise<channel.Channel>]>([
      ["an index", createIndex],
      ["a virtual", createVirtual],
      ["a calculated", createCalculated],
    ])("should show the row for %s channel", async (_, create) => {
      const ch = await create();
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
      const tooltip = await findTooltip();
      expect(within(tooltip).queryByText(LABEL)).not.toBeNull();
    });

    it("should omit the row when no channel is set", async () => {
      renderTooltip(Node.createConfig({ variant: "value" }));
      const tooltip = await findTooltip();
      expect(within(tooltip).queryByText(LABEL)).toBeNull();
    });

    describe("config changes while open", () => {
      const createSampled = async (): Promise<channel.Channel> => {
        const ch = await createChannel();
        stamps.stamp(ch.key, TimeStamp.now());
        return ch;
      };

      // Cached, so a swap re-renders the open tooltip instead of remounting it.
      const createCached = async (): Promise<channel.Channel> => {
        const ch = await createChannel();
        await client.channels.retrieve(ch.key);
        return ch;
      };

      const renderSwappable = async (config: Node.Config) => {
        const anchor = createAnchor();
        const { rerender } = render(<Tooltip anchor={anchor} config={config} />, {
          wrapper: Wrapper,
        });
        const tooltip = await findTooltip();
        await findRowValue(tooltip);
        const label = labelOf(tooltip, LABEL);
        const swap = (next: Node.Config): void => {
          rerender(<Tooltip anchor={anchor} config={next} />);
          expect(getTooltip()).toBe(tooltip);
        };
        return { tooltip, label, swap };
      };

      it("should follow a new channel", async () => {
        const [a, b] = await Promise.all([createSampled(), createCached()]);
        const { tooltip, label, swap } = await renderSwappable(
          Node.createConfig({ variant: "value", channel: a.key }),
        );
        swap(Node.createConfig({ variant: "value", channel: b.key }));
        expect(labelOf(tooltip, LABEL)).not.toBe(label);
        await sleep.sleep(TimeSpan.milliseconds(250));
        expect(valueOf(labelOf(tooltip, LABEL)).textContent).toEqual("");
      });

      it("should follow a state channel added to a command-only valve", async () => {
        const [command, state] = await Promise.all([createSampled(), createCached()]);
        const { tooltip, label, swap } = await renderSwappable(
          Node.createConfig({ variant: "valve", commandChannel: command.key }),
        );
        swap(
          Node.createConfig({
            variant: "valve",
            commandChannel: command.key,
            stateChannel: state.key,
          }),
        );
        expect(labelOf(tooltip, LABEL)).not.toBe(label);
        await sleep.sleep(TimeSpan.milliseconds(250));
        expect(valueOf(labelOf(tooltip, LABEL)).textContent).toEqual("");
      });

      it("should keep the row when only another field changes", async () => {
        const a = await createSampled();
        const { tooltip, label, swap } = await renderSwappable(
          Node.createConfig({ variant: "value", channel: a.key }),
        );
        swap(
          Node.createConfig({
            variant: "value",
            channel: a.key,
            stalenessTimeout: TimeSpan.seconds(10).nanoseconds,
          }),
        );
        expect(labelOf(tooltip, LABEL)).toBe(label);
        expect(valueOf(label).textContent).not.toEqual("");
      });

      it("should drop the row when the channel is cleared", async () => {
        const a = await createSampled();
        const { tooltip, swap } = await renderSwappable(
          Node.createConfig({ variant: "value", channel: a.key }),
        );
        swap(Node.createConfig({ variant: "value" }));
        expect(within(tooltip).queryByText(LABEL)).toBeNull();
      });
    });
  });

  describe("dividers", () => {
    it("should divide channel rows, field rows, and the sample age row", async () => {
      const ch = await createChannel();
      renderTooltip(Node.createConfig({ variant: "value", channel: ch.key }));
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
