// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Tooltip.css";

import { channel } from "@synnaxlabs/client";
import { Description } from "@synnaxlabs/lyra/description";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Tooltip as Base } from "@synnaxlabs/lyra/tooltip";
import { caseconv, type color, primitive, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useEffect, useState } from "react";

import { Channel } from "@/channel";
import { CSS } from "@/css";
import { type Node } from "@/schematic/node";
import { Synnax } from "@/synnax";
import { LatestSample } from "@/vis/latestSample";
import { Staleness } from "@/vis/staleness";

export interface TooltipProps {
  anchor: HTMLElement;
  config: Node.Config;
}

const LOCATION: Base.FrameProps["location"] = { x: "center", y: "bottom" };
const LIST_CLASS = CSS.BE("schematic-tooltip", "list");

type KeysOf<C> = C extends object ? keyof C : never;
type Field = KeysOf<Node.Config>;

const CHANNEL_ROWS: { field: Field; icon: Icon.FC }[] = [
  { field: "stateChannel", icon: Icon.Channel },
  { field: "channel", icon: Icon.Channel },
  { field: "commandChannel", icon: Icon.Control },
];

// Channels the symbol already streams come first, so the row adds no subscription.
const SAMPLE_AGE_ORDER: Field[] = ["stateChannel", "channel", "commandChannel"];

const FIELD_ROWS: { field: Field; label?: string }[] = [
  { field: "mode" },
  { field: "normallyOpen" },
  { field: "clickable" },
  { field: "onClickDelay" },
  { field: "stalenessTimeout", label: "Stale timeout" },
];

const SPAN_FIELDS = new Set<Field>(["onClickDelay", "stalenessTimeout"]);

const kindIcon = (ch: channel.Channel): Icon.FC | null => {
  if (ch.isIndex) return Icon.Time;
  if (channel.isCalculated(ch.payload)) return Icon.Calculation;
  if (ch.virtual) return Icon.Virtual;
  return null;
};

interface RowProps {
  label: ReactNode;
  value: ReactNode;
  labelColor?: Theming.Shade;
  valueColor?: Theming.Shade | color.Crude;
  valueClassName?: string;
}

const Row = ({
  label,
  value,
  labelColor,
  valueColor,
  valueClassName,
}: RowProps): ReactElement => (
  <Description.Item>
    <Description.Label color={labelColor}>{label}</Description.Label>
    <Description.Value variant="code" color={valueColor} className={valueClassName}>
      {value}
    </Description.Value>
  </Description.Item>
);

interface SampleAgeProps {
  channel: channel.Key;
}

// Mounted only once the tooltip shows, so a mouse sweep creates no worker component.
const SampleAge = ({ channel }: SampleAgeProps): ReactElement => {
  const time = LatestSample.use({ channel });
  // The hook counts on the local clock, so the stamp moves by the measured skew.
  const { clockSkew } = Synnax.useConnectionStatus().details;
  const since = Telem.Text.useTimeSpanSince(time?.add(clockSkew) ?? 0).toString(
    "semantic",
  );
  return (
    <Description.List level="small" justify="between" className={LIST_CLASS}>
      <Row
        valueClassName={CSS.BE("schematic-tooltip", "sample-age")}
        label={
          <>
            <Icon.TimeOutline />
            Sample age
          </>
        }
        value={time == null ? undefined : since}
      />
    </Description.List>
  );
};

/** Shows a symbol's configuration beside its element. */
export const Tooltip = ({ anchor, config }: TooltipProps): ReactElement | null => {
  const { delay, isWarm, markClosed } = Base.useConfig();
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    if (isWarm()) return setVisible(true);
    const timeout = setTimeout(
      () => setVisible(true),
      new TimeSpan(delay).milliseconds,
    );
    return () => clearTimeout(timeout);
  }, []);
  useEffect(() => (visible ? markClosed : undefined), [visible]);
  const theme = Theming.use();
  const values: Partial<Record<Field, primitive.Value>> = config;
  const keys = CHANNEL_ROWS.flatMap(({ field }) => {
    const key = values[field];
    return typeof key === "number" && !primitive.isZero(key) ? key : [];
  });
  // A null query holds the fetch until the delay passes, so mouse sweeps fire nothing.
  const { data } = Channel.useResultMultiple(
    visible && keys.length > 0 ? { where: { key: { in: keys } } } : null,
  );
  if (!visible || (keys.length > 0 && data == null)) return null;
  const channels = CHANNEL_ROWS.flatMap(({ field, icon: RoleIcon }) => {
    const ch = data?.find((c) => c.key === values[field]);
    if (ch == null) return [];
    const KindIcon = kindIcon(ch);
    return (
      <Row
        key={field}
        labelColor={10}
        label={
          <>
            <RoleIcon />
            {ch.name}
          </>
        }
        value={
          <>
            {KindIcon != null && <KindIcon />}
            {ch.dataType.toString(true)}
          </>
        }
      />
    );
  });
  const followed = SAMPLE_AGE_ORDER.map((field) =>
    data?.find((c) => c.key === values[field]),
  ).find((c) => c != null);
  const stalenessColor = Staleness.resolveColor(
    "stalenessColor" in config ? config.stalenessColor : undefined,
    theme,
  );
  const fields = FIELD_ROWS.flatMap(({ field, label }) => {
    const value = values[field];
    if (value == null) return [];
    // A zero click delay means none, so the row would only be noise.
    if (field === "onClickDelay" && value === 0) return [];
    const text = SPAN_FIELDS.has(field)
      ? new TimeSpan(Number(value)).toString("full")
      : String(value);
    return (
      <Row
        key={field}
        label={label ?? caseconv.toSentence(field)}
        value={text}
        valueColor={field === "stalenessTimeout" ? stalenessColor : undefined}
      />
    );
  });
  if (channels.length + fields.length === 0) return null;
  const divider = (
    <Divider.Divider x className={CSS.BE("schematic-tooltip", "divider")} />
  );
  return (
    <Base.Frame
      anchor={anchor}
      location={LOCATION}
      className={CSS.B("schematic-tooltip")}
    >
      {channels.length > 0 && (
        <Description.List level="small" justify="between" className={LIST_CLASS}>
          {channels}
        </Description.List>
      )}
      {channels.length > 0 && fields.length > 0 && divider}
      {fields.length > 0 && (
        <Description.List level="small" justify="between" className={LIST_CLASS}>
          {fields}
        </Description.List>
      )}
      {followed != null && (
        <>
          {divider}
          <SampleAge key={followed.key} channel={followed.key} />
        </>
      )}
    </Base.Frame>
  );
};
