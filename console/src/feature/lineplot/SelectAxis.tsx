// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot, type ranger } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, LinePlot } from "@synnaxlabs/pluto";
import { dynamicWindow, TimeSpan, unique } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CustomRangeInput } from "@/feature/lineplot/CustomRangeInput";
import { CSS } from "@/platform/css";
import { Range } from "@/platform/range";
import { Session } from "@/session";

const SEARCH_OPTIONS: channel.RetrieveOptions = {
  notDataTypes: ["string", "json", "uuid"],
  internal: false,
  virtual: false,
};

export interface YAxisChannelSelectProps extends Omit<
  Input.ItemProps,
  "onChange" | "children" | "label"
> {
  axisKey: lineplot.YAxisKey;
}

export const YAxisChannelSelect = ({
  axisKey,
  ...rest
}: YAxisChannelSelectProps): ReactElement => {
  const value = LinePlot.useYAxisChannels({ axisKey });
  const dispatch = LinePlot.useSingleDispatch();
  const rangeKey = Session.Range.useSelectSelectedKey();
  const handleChange = (channels: channel.Key[]) =>
    dispatch(lineplot.setChannels({ axisKey, channels }));
  return (
    <Input.Item x label={LinePlot.axisLabel(axisKey)} {...rest}>
      <Channel.SelectMultiple
        value={value}
        initialQuery={{ ...SEARCH_OPTIONS, rangeKey }}
        onChange={handleChange}
        full="x"
        location="top"
      />
    </Input.Item>
  );
};

export interface XAxisChannelSelectProps extends Omit<
  Input.ItemProps,
  "onChange" | "children" | "label"
> {
  axisKey: lineplot.XAxisKey;
}

const TRIGGER_PROPS = { placeholder: "Time", auto: true };

interface TimeOptionProps {
  selected: boolean;
  onSelect: () => void;
}

// Pinned under the list, out of its search and arrow keys, so it stays in reach.
const TimeOption = ({ selected, onSelect }: TimeOptionProps): ReactElement => {
  const { close } = Dialog.useContext();
  const handleChange = () => {
    onSelect();
    close();
  };
  return (
    <Flex.Box className={CSS.BE("line-plot", "x-time")} full="x">
      <Button.Toggle
        value={selected}
        onChange={handleChange}
        variant="text"
        full="x"
        justify="start"
        className={CSS.BE("line-plot", "x-time-btn")}
      >
        <Flex.Box y gap="tiny" align="start" full="x">
          <Text.Text className={CSS.BE("line-plot", "x-time-title")}>
            <Icon.Time />
            Time
            <Input.AutoChip className={CSS.BE("line-plot", "x-time-chip")} />
          </Text.Text>
          <Text.Text level="small" color={9}>
            Automatically uses each channel's time. Best for most needs.
          </Text.Text>
        </Flex.Box>
      </Button.Toggle>
    </Flex.Box>
  );
};

export const XAxisChannelSelect = ({
  axisKey,
  ...rest
}: XAxisChannelSelectProps): ReactElement => {
  const value = LinePlot.useXAxisChannel({ axisKey });
  const dispatch = LinePlot.useSingleDispatch();
  const rangeKey = Session.Range.useSelectSelectedKey();
  const handleChange = (channel: channel.Key | null) =>
    dispatch(lineplot.setXChannel({ axisKey, channel: channel ?? 0 }));
  return (
    <Input.Item x label={LinePlot.axisLabel(axisKey)} {...rest} grow>
      <Channel.SelectSingle
        value={value}
        onChange={handleChange}
        allowNone
        initialQuery={{ ...SEARCH_OPTIONS, rangeKey }}
        location="top"
        icon={<Icon.Time />}
        footer={<TimeOption selected={value === 0} onSelect={() => handleChange(0)} />}
        triggerProps={TRIGGER_PROPS}
      />
    </Input.Item>
  );
};

export interface RangeSelectProps extends Omit<
  Range.SelectMultipleInputItemProps,
  "value" | "onChange"
> {}

// The window that selecting the custom entry adds. No session range has this span, so
// the plot keeps showing it as the custom entry.
const DEFAULT_CUSTOM: ranger.Ref = {
  variant: "window",
  window: dynamicWindow(TimeSpan.hours(2)),
};

// Two session ranges can show the same window, and a plot draws each reference once.
const dedupe = (refs: ranger.Ref[]): ranger.Ref[] =>
  refs.filter(
    (r, i) => refs.findIndex((other) => Session.Range.refsEqual(other, r)) === i,
  );

/**
 * Selects the ranges the plot shows. A window that no session range shows is the
 * custom entry, which reveals an input to edit the window.
 */
export const RangeSelect = (props: RangeSelectProps): ReactElement => {
  const refs = LinePlot.useRanges();
  const states = Session.Range.useSelectMultiple();
  const dispatch = LinePlot.useSingleDispatch();
  const keyOf = (ref: ranger.Ref): string =>
    ref.variant === "key"
      ? ref.key
      : (Session.Range.findKey(ref, states) ?? Range.CUSTOM_KEY);
  const customs = refs.filter((r) => keyOf(r) === Range.CUSTOM_KEY);
  const value = unique.unique(refs.map(keyOf));
  const handleChange = (keys: string[]) => {
    const ranges = keys.flatMap((k): ranger.Ref[] => {
      if (k === Range.CUSTOM_KEY)
        return customs.length > 0 ? customs : [DEFAULT_CUSTOM];
      const state = states.find((s) => s.key === k);
      return [state == null ? { variant: "key", key: k } : Session.Range.toRef(state)];
    });
    dispatch(lineplot.setRanges({ ranges: dedupe(ranges) }));
  };
  const custom = customs.find(
    (r): r is ranger.WindowRef =>
      r.variant === "window" && r.window.variant === "dynamic",
  );
  const handleCustomChange = (span: TimeSpan) =>
    dispatch(
      lineplot.setRanges({
        ranges: dedupe(
          refs.map((r) =>
            r === custom ? { variant: "window", window: dynamicWindow(span) } : r,
          ),
        ),
      }),
    );
  return (
    <Flex.Box x grow>
      <Range.SelectMultipleInputItem value={value} onChange={handleChange} {...props} />
      {custom?.window.variant === "dynamic" && (
        <CustomRangeInput
          value={new TimeSpan(custom.window.timeSpan)}
          onChange={handleCustomChange}
        />
      )}
    </Flex.Box>
  );
};
