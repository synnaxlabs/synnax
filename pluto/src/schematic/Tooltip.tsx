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
import { caseconv, type color, primitive, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useEffect, useState } from "react";

import { Channel } from "@/channel";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type Node } from "@/schematic/node";
import { Telem } from "@/telem";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Tooltip as Base } from "@/tooltip";
import { LatestSample } from "@/vis/latestSample";
import { Staleness } from "@/vis/staleness";

export interface TooltipProps {
  anchor: HTMLElement;
  config: Node.Config;
}

const LOCATION: Base.FrameProps["location"] = { x: "center", y: "bottom" };

type KeysOf<C> = C extends object ? keyof C : never;
type Field = KeysOf<Node.Config>;

const CHANNEL_ROWS: { field: Field; icon: Icon.FC }[] = [
  { field: "commandChannel", icon: Icon.Edit },
  { field: "stateChannel", icon: Icon.VisibleFilled },
  { field: "channel", icon: Icon.VisibleFilled },
];

// Channels the symbol already streams come first, so the row adds no subscription.
const LAST_WRITE_ORDER: Field[] = ["stateChannel", "channel", "commandChannel"];

const FIELD_ROWS: { field: Field; unit?: string }[] = [
  { field: "mode" },
  { field: "normallyOpen" },
  { field: "clickable" },
  { field: "onClickDelay", unit: "ms" },
  { field: "stalenessTimeout" },
];

const kindIcon = (ch: channel.Channel): Icon.FC | null => {
  if (channel.isCalculated(ch.payload)) return Icon.Calculation;
  if (ch.virtual) return Icon.Virtual;
  return null;
};

const hasLastWrite = (ch: channel.Channel): boolean =>
  (ch.isIndex || !primitive.isZero(ch.index)) && !channel.isCalculated(ch.payload);

interface RowProps {
  label: ReactNode;
  value: ReactNode;
  color?: Theming.Shade | color.Crude;
  className?: string;
}

const Row = ({ label, value, color, className }: RowProps): ReactElement => (
  <Flex.Box x justify="between" gap="large" className={className}>
    <Text.Text level="small">{label}</Text.Text>
    <Text.Text
      level="small"
      color={color}
      className={CSS.BE("schematic-tooltip", "value")}
    >
      {value}
    </Text.Text>
  </Flex.Box>
);

interface LastWriteProps {
  name: string;
  channel: channel.Key;
}

// Mounted only once the tooltip shows, so a mouse sweep creates no worker component.
const LastWrite = ({ name, channel }: LastWriteProps): ReactElement => {
  const time = LatestSample.use({ channel });
  const since = Telem.Text.useTimeSpanSince(time ?? 0);
  return (
    <Row
      label={
        <>
          <Icon.Time />
          {name}
        </>
      }
      value={time == null ? undefined : since.toString("semantic")}
    />
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
    visible && keys.length > 0 ? { keys } : null,
  );
  const followed = LAST_WRITE_ORDER.map((field) =>
    data?.find((c) => c.key === values[field]),
  ).find((c) => c != null && hasLastWrite(c));
  const indexKey =
    followed == null ? null : followed.isIndex ? followed.key : followed.index;
  const index = Channel.useResult(indexKey == null ? null : { key: indexKey });
  if (!visible || (keys.length > 0 && data == null) || index.variant === "loading")
    return null;
  const channels = CHANNEL_ROWS.flatMap(({ field, icon: RoleIcon }) => {
    const ch = data?.find((c) => c.key === values[field]);
    if (ch == null || ch.isIndex) return [];
    const KindIcon = kindIcon(ch);
    return (
      <Row
        key={field}
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
  if (followed != null && index.data != null)
    channels.push(
      <LastWrite key="lastWrite" name={index.data.name} channel={followed.key} />,
    );
  const stalenessColor = Staleness.resolveColor(
    "stalenessColor" in config ? config.stalenessColor : undefined,
    theme,
  );
  const fields = FIELD_ROWS.flatMap(({ field, unit }) => {
    const value = values[field];
    if (value == null) return [];
    // A zero click delay means none, so the row would only be noise.
    if (field === "onClickDelay" && value === 0) return [];
    const text =
      field === "stalenessTimeout"
        ? TimeSpan.seconds(Number(value)).toString("semantic")
        : `${String(value)}${unit ?? ""}`;
    return (
      <Row
        key={field}
        className={CSS.BE("schematic-tooltip", "field")}
        label={caseconv.toSentence(field)}
        value={text}
        color={field === "stalenessTimeout" ? stalenessColor : undefined}
      />
    );
  });
  if (channels.length + fields.length === 0) return null;
  return (
    <Base.Frame
      anchor={anchor}
      location={LOCATION}
      className={CSS.B("schematic-tooltip")}
    >
      {channels}
      {channels.length > 0 && fields.length > 0 && (
        <Divider.Divider x className={CSS.BE("schematic-tooltip", "divider")} />
      )}
      {fields}
    </Base.Frame>
  );
};
