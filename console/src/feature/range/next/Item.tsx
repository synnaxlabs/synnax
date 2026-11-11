// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/next/Range.css";

import { type label, ranger } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { context } from "@synnaxlabs/lyra/context";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { type position } from "@synnaxlabs/lyra/position";
import { Select } from "@synnaxlabs/lyra/select";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Access, Label as PLabel, Ranger } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type MouseEvent,
  type ReactElement,
  type ReactNode,
} from "react";

import { useNode } from "@/feature/range/next/tree";
import { CSS } from "@/platform/css";
import { Label } from "@/platform/label";
import { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Session } from "@/session";

export const [HighlightContext, useHighlight] = context.create<string | undefined>({
  defaultValue: undefined,
  displayName: "Range.HighlightContext",
});

export const useOpen = (): ((key: ranger.Key) => void) => {
  const openTab = Panel.useOpenTab();
  return (key) => openTab({ variant: "resource", resource: ranger.resourceID(key) });
};

export const useStop = (): ((range: ranger.Range) => void) => {
  const { update } = Ranger.useSetTimeRange();
  return (range) =>
    update({
      key: range.key,
      timeRange: Ranger.moveToStage(range.timeRange.numeric, "completed"),
    });
};

interface CaretProps {
  range: ranger.Range;
  expanded: boolean;
  onExpand: (key: ranger.Key, expanded: boolean) => void;
}

const Caret = ({ range, expanded, onExpand }: CaretProps): ReactElement => {
  if (!range.hasChildren) return <span className={CSS.BE("range-item", "caret")} />;
  const handleClick = (e: MouseEvent) => {
    e.stopPropagation();
    onExpand(range.key, !expanded);
  };
  return (
    <Button.Button
      variant="text"
      size="tiny"
      textColor={8}
      className={CSS.cls(
        CSS.BE("range-item", "caret"),
        expanded && CSS.BEM("range-item", "caret", "expanded"),
      )}
      aria-label={expanded ? "Collapse" : "Expand"}
      aria-expanded={expanded}
      onClick={handleClick}
    >
      <Icon.Caret.Right />
    </Button.Button>
  );
};

const Path = ({ parent }: { parent: ranger.Key }): ReactElement | null => {
  const range = Ranger.useResult({ key: parent }).data;
  if (range == null) return null;
  return (
    <Text.Text
      level="small"
      color={8}
      overflow="fade"
      className={CSS.BE("range-item", "path")}
    >
      {range.name}
    </Text.Text>
  );
};

const stopPropagation = (e: MouseEvent) => e.stopPropagation();

interface FieldProps {
  className: string;
  children: ReactNode;
}

/** Holds a control in an item, so a click on it neither selects nor opens the item. */
const Field = ({ className, children }: FieldProps): ReactElement => (
  <span className={className} onClick={stopPropagation} onDoubleClick={stopPropagation}>
    {children}
  </span>
);

/** The color of a range. A range with no color shows its swatch on hover only. */
const Swatch = ({ range }: { range: ranger.Range }): ReactElement => {
  const canEdit = Access.useUpdateGranted(ranger.resourceID(range.key));
  const { update } = Ranger.useSetColor();
  const { colors } = Theming.use();
  return (
    <Field className={CSS.BE("range-item", "swatch")}>
      {(canEdit || range.color != null) && (
        <Color.Swatch
          variant="shadow"
          size="tiny"
          value={range.color}
          fallback={colors.gray.l8}
          reveal={range.color == null}
          onChange={canEdit ? (color) => update({ key: range.key, color }) : undefined}
          onlyChangeOnBlur
          tooltip="Color"
        />
      )}
    </Field>
  );
};

/** The labels grow to the left, so their right edge holds the picker still. */
const RIGHT_EDGE: position.LocationPreference = {
  targetCorner: { x: "right" },
  dialogCorner: { x: "right" },
};

/** The labels of a range. Clicking them opens the label picker. */
const Labels = ({ range }: { range: ranger.Range }): ReactElement | null => {
  const id = ranger.resourceID(range.key);
  const canEdit = Access.useUpdateGranted(id);
  const { update } = PLabel.useEditFor();
  if (!canEdit)
    return range.labels.length === 0 ? null : (
      <Tag.Tags variant="text" className={CSS.BE("range-item", "labels")}>
        <PLabel.Tags keys={range.labels} size="tiny" />
      </Tag.Tags>
    );
  const handleChange = (next: label.Key[]): void =>
    update({
      id,
      added: next.filter((key) => !range.labels.includes(key)),
      removed: range.labels.filter((key) => !next.includes(key)),
    });
  return (
    <Field className={CSS.BE("range-item", "labels")}>
      <Label.SelectMultiple
        zIndex={100}
        location={RIGHT_EDGE}
        value={range.labels}
        onChange={handleChange}
        triggerProps={{ variant: "shadow" }}
      />
    </Field>
  );
};

export interface LeadProps {
  range: ranger.Range;
  depth?: number;
  expanded?: boolean;
  /** Expands the range. Without it, the range shows no caret and no space for one. */
  onExpand?: (key: ranger.Key, expanded: boolean) => void;
  /** Shows the name of the parent after the name, for a range out of its tree. */
  showPath?: boolean;
}

/** The caret, color, and name of a range, shared by the list and the timeline. */
export const Lead = ({
  range,
  depth = 0,
  expanded = false,
  onExpand,
  showPath = false,
}: LeadProps): ReactElement => {
  const canEdit = Access.useUpdateGranted(ranger.resourceID(range.key));
  const { update: rename } = Ranger.useRename();
  const highlight = useHighlight();
  return (
    <>
      <span
        className={CSS.BE("range-item", "indent")}
        style={{ "--console-range-depth": depth } as CSSProperties}
      />
      {onExpand != null && (
        <Caret range={range} expanded={expanded} onExpand={onExpand} />
      )}
      <Swatch range={range} />
      <Flex.Box x align="center" gap="medium" className={CSS.BE("range-item", "name")}>
        <Text.MaybeEditable
          id={List.itemNameID(range.key)}
          value={range.name}
          overflow="fade"
          allowDoubleClick={false}
          highlight={highlight}
          onChange={canEdit ? (name) => rename({ key: range.key, name }) : undefined}
        />
        {showPath && range.parent != null && <Path parent={range.parent} />}
      </Flex.Box>
    </>
  );
};

/** The start of a range and how long it ran, or how long it has run so far. */
const When = ({ range }: { range: ranger.Range }): ReactElement => {
  const now = Ranger.useClock(range.timeRange);
  const { start, end } = range.timeRange.makeValid();
  const stage = Ranger.getStage(range.timeRange, now);
  const day = Telem.Text.describeDay(start, now);
  let span: ReactElement;
  if (stage === "in_progress")
    span = (
      <Flex.Box
        x
        align="center"
        gap="small"
        className={CSS.BE("range-item", "span")}
        aria-label="Running"
      >
        <Icon.InProgress />
        <Telem.Text.TimeSpanSince level="small" format="semantic" color={10}>
          {start}
        </Telem.Text.TimeSpanSince>
      </Flex.Box>
    );
  else if (stage === "to_do")
    span = (
      <Flex.Box
        x
        align="center"
        gap="small"
        className={CSS.BE("range-item", "span")}
        aria-label="Planned"
      >
        <Icon.ToDo color={9} />
        <Text.Text level="small" color={9}>
          Planned
        </Text.Text>
      </Flex.Box>
    );
  else
    span = (
      <Text.Text level="small" color={8} className={CSS.BE("range-item", "span")}>
        {start.span(end).toString("semantic")}
      </Text.Text>
    );
  return (
    <Flex.Box
      x
      align="center"
      gap="small"
      role="group"
      aria-label="Time range"
      className={CSS.BE("range-item", "when")}
    >
      <Text.Text level="small" color={9} className={CSS.BE("range-item", "start")}>
        {`${day} ${Telem.Text.formatTime(start, TimeSpan.MINUTE)}`}
      </Text.Text>
      {span}
    </Flex.Box>
  );
};

export interface ItemProps extends List.ItemProps<ranger.Key> {
  depth?: number;
  expanded?: boolean;
  onExpand?: (key: ranger.Key, expanded: boolean) => void;
  /** Shows the name of the parent after the name, for a range out of its tree. */
  showPath?: boolean;
}

/** A range as a row: its lead, labels, time, and favorite button. */
export const Item = ({
  depth,
  expanded,
  onExpand,
  showPath,
  className,
  ...rest
}: ItemProps): ReactElement | null => {
  const { itemKey } = rest;
  const range = List.useItem<ranger.Key, ranger.Range>(itemKey);
  const pinned = Session.Range.useSelectState(itemKey) != null;
  const open = useOpen();
  if (range == null) return null;
  return (
    <Select.Item<ranger.Key>
      {...rest}
      className={CSS.cls(CSS.B("range-item"), className)}
      aria-label={range.name}
      checkable
      onActivate={() => open(itemKey)}
    >
      <Lead
        range={range}
        depth={depth}
        expanded={expanded}
        onExpand={onExpand}
        showPath={showPath}
      />
      <Labels range={range} />
      <When range={range} />
      <Range.FavoriteButton
        range={range}
        reveal
        size="small"
        tooltip={pinned ? "Unpin" : "Pin"}
        className={CSS.BE("range-item", "favorite")}
      />
    </Select.Item>
  );
};

/** An item in a tree, at its depth, with a caret that opens its children. */
export const TreeItem = (props: List.ItemProps<ranger.Key>): ReactElement => {
  const { depth, expanded, onExpand } = useNode(props.itemKey);
  return (
    <Item
      {...props}
      role="listitem"
      depth={depth}
      expanded={expanded}
      onExpand={onExpand}
    />
  );
};

/** An item out of its tree, with the name of its parent. */
export const FlatItem = (props: List.ItemProps<ranger.Key>): ReactElement => (
  <Item {...props} role="listitem" showPath />
);
