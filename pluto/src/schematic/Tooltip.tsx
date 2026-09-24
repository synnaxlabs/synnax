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
import { caseconv, type color, type primitive, TimeSpan } from "@synnaxlabs/x";
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
  { field: "stateChannel", icon: Icon.Visible },
  { field: "channel", icon: Icon.Visible },
];

const FIELD_ROWS: { field: Field; unit?: string }[] = [
  { field: "mode" },
  { field: "normallyOpen" },
  { field: "clickable" },
  { field: "onClickDelay", unit: "ms" },
  { field: "stalenessTimeout", unit: "s" },
];

const kindIcon = (ch: channel.Channel): Icon.FC | null => {
  if (channel.isCalculated(ch.payload)) return Icon.Calculation;
  if (ch.virtual) return Icon.Virtual;
  return null;
};

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
      variant="code"
      color={color}
      className={CSS.BE("schematic-tooltip", "value")}
    >
      {value}
    </Text.Text>
  </Flex.Box>
);

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
    return typeof key === "number" && key !== 0 ? key : [];
  });
  // A null query holds the fetch until the delay passes, so mouse sweeps fire nothing.
  const { data } = Channel.useResultMultiple(
    visible && keys.length > 0 ? { keys } : null,
  );
  const indexed = data?.find(
    (c) => c.index !== 0 && !channel.isCalculated(c.payload),
  );
  const indexKey = indexed?.index ?? null;
  const index = Channel.useResult(indexKey == null ? null : { key: indexKey });
  const lastSample = LatestSample.use({ channel: indexKey ?? 0 });
  const sinceLastSample = Telem.Text.useTimeSpanSince(lastSample ?? 0);
  if (
    !visible ||
    (keys.length > 0 && data == null) ||
    index.variant === "loading" ||
    (indexKey != null && lastSample === undefined)
  )
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
  const stalenessColor = Staleness.resolveColor(
    "stalenessColor" in config ? config.stalenessColor : undefined,
    theme,
  );
  const fields = FIELD_ROWS.flatMap(({ field, unit }) => {
    const value = values[field];
    if (value == null) return [];
    return (
      <Row
        key={field}
        className={CSS.BE("schematic-tooltip", "field")}
        label={caseconv.toSentence(field)}
        value={unit == null ? String(value) : `${String(value)}${unit}`}
        color={field === "stalenessTimeout" ? stalenessColor : undefined}
      />
    );
  });
  const lastWrite =
    lastSample == null
      ? undefined
      : sinceLastSample.toString("semantic");
  if (index.data != null)
    channels.push(
      <Row
        key="index"
        label={
          <>
            <Icon.Time />
            {index.data.name}
          </>
        }
        value={lastWrite}
      />,
    );
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
