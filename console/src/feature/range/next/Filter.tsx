// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/next/Filter.css";

import { type label, ranger } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { type position } from "@synnaxlabs/lyra/position";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Flux, Label, Ranger } from "@synnaxlabs/pluto";
import { deep, filter as xfilter, location, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import {
  type Condition,
  type ConditionOf,
  decode,
  drop,
  encode,
  find,
  findPreset,
  type LabelMatch,
  type Property,
  put,
  TIME_PRESETS,
} from "@/feature/range/next/condition";
import { type MenuProperty, useSubmenus } from "@/feature/range/next/submenus";
import { CSS } from "@/platform/css";
import { View } from "@/platform/view";

const path = deep.path<{ where?: ranger.Where }>();

const useConditions = (): [Condition[], (next: Condition[]) => void] => {
  const where = Form.useFieldValue<ranger.Where | undefined>(path.where, {
    optional: true,
  });
  const { set } = Form.useContext();
  return [decode(where ?? undefined), (next) => set(path.where, encode(next))];
};

const BESIDE: position.LocationPreference = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
};

const BELOW: position.LocationPreference = {
  targetCorner: location.BOTTOM_LEFT,
  dialogCorner: location.TOP_LEFT,
};

interface PickerProps extends Pick<Dialog.FrameProps, "visible" | "onVisibleChange"> {
  /** The trigger that opens the picker. */
  trigger: ReactNode;
  location: position.LocationPreference;
  /** Names the surface of the picker, so the menu can find it. */
  surfaceID?: string;
  /** Extra classes for the surface of the picker. */
  className?: string;
}

/** The option for ranges with no labels. Only the picker holds this key. */
const NO_LABELS = "no-labels";

const LabelItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<label.Key>): ReactElement | null => {
  const item = List.useItem<label.Key, label.Label>(itemKey);
  if (item == null) return null;
  return (
    <Select.Item<label.Key> itemKey={itemKey} {...rest}>
      <Icon.Circle color={item.color} size="2.5em" />
      {item.name}
    </Select.Item>
  );
};

const labelItem = Component.renderProp(LabelItem);

const pickerClass = (className?: string, search = false): string =>
  CSS.cls(
    CSS.BE("range-filter", "picker"),
    search && CSS.BEM("range-filter", "picker", "search"),
    className,
  );

const LabelPicker = ({
  trigger,
  location,
  surfaceID,
  className,
  ...frame
}: PickerProps): ReactElement => {
  const [conditions, setConditions] = useConditions();
  const condition = find(conditions, "labels");
  let value: label.Key[] = [];
  if (condition != null)
    value = condition.match === "empty" ? [NO_LABELS] : condition.keys;
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client) => client.labels,
    { name: "labels" },
  );
  const handleChange = (next: label.Key[]): void => {
    if (next.includes(NO_LABELS) && !value.includes(NO_LABELS)) {
      setConditions(put(conditions, { property: "labels", match: "empty", keys: [] }));
      return;
    }
    const keys = next.filter((key) => key !== NO_LABELS);
    if (keys.length === 0) {
      setConditions(drop(conditions, "labels"));
      return;
    }
    const match =
      condition == null || condition.match === "empty" ? "any" : condition.match;
    setConditions(put(conditions, { property: "labels", match, keys }));
  };
  return (
    <Dialog.Frame variant="floating" location={location} {...frame}>
      <Select.Frame<label.Key, label.Label | undefined>
        multiple
        value={value}
        onChange={handleChange}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={more}
      >
        {trigger}
        <Select.Dialog
          surface="menu"
          data-surface={surfaceID}
          className={pickerClass(className, true)}
        >
          <Select.Body<label.Key>
            resourceName="label"
            onSearch={search}
            status={status}
            fixedItems={
              <Select.Item<label.Key> itemKey={NO_LABELS} textValue="No labels">
                <Icon.Label />
                No labels
              </Select.Item>
            }
          >
            {labelItem}
          </Select.Body>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

const StagePicker = ({
  trigger,
  location,
  surfaceID,
  className,
  ...frame
}: PickerProps): ReactElement => {
  const [conditions, setConditions] = useConditions();
  const value = find(conditions, "stage")?.stages ?? [];
  const handleChange = (stages: ranger.Stage[]): void =>
    setConditions(
      stages.length === 0
        ? drop(conditions, "stage")
        : put(conditions, { property: "stage", stages }),
    );
  return (
    <Dialog.Frame variant="floating" location={location} {...frame}>
      <Select.Frame<ranger.Stage, undefined>
        multiple
        value={value}
        onChange={handleChange}
      >
        {trigger}
        <Select.Dialog
          surface="menu"
          data-surface={surfaceID}
          className={pickerClass(className)}
        >
          <Select.List>
            {ranger.STAGES.map((stage) => {
              const StageIcon = Ranger.STAGE_ICONS[stage];
              return (
                <Select.Item<ranger.Stage> key={stage} itemKey={stage}>
                  <StageIcon />
                  {Ranger.STAGE_NAMES[stage]}
                </Select.Item>
              );
            })}
          </Select.List>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

const TimePicker = ({
  trigger,
  location,
  surfaceID,
  className,
  ...frame
}: PickerProps): ReactElement => {
  const [conditions, setConditions] = useConditions();
  const window = find(conditions, "time")?.window;
  const preset = window instanceof TimeSpan ? findPreset(window)?.key : undefined;
  const setWindow = (next: TimeSpan | TimeRange): void =>
    setConditions(put(conditions, { property: "time", window: next }));
  const handlePreset = (key: string | null): void => {
    const next = TIME_PRESETS.find((p) => p.key === key);
    if (next == null) setConditions(drop(conditions, "time"));
    else setWindow(next.span);
  };
  const now = useNow(() => null);
  const range =
    window instanceof TimeRange
      ? window
      : new TimeRange(now.sub(window ?? TimeSpan.days(7)), now);
  return (
    <Dialog.Frame variant="floating" location={location} {...frame}>
      <Select.Frame<string, undefined> value={preset} onChange={handlePreset} allowNone>
        {trigger}
        <Select.Dialog
          surface="menu"
          data-surface={surfaceID}
          className={pickerClass(className)}
        >
          <Select.List>
            {TIME_PRESETS.map(({ key, name }) => (
              <Select.Item<string> key={key} itemKey={key}>
                {name}
              </Select.Item>
            ))}
          </Select.List>
          <Flex.Box y gap="small" className={CSS.BE("range-filter", "custom")}>
            <Text.Text level="small" color={9}>
              Custom range
            </Text.Text>
            <Input.Item x label="From" className={CSS.BE("range-filter", "bound")}>
              <Input.DateTime
                value={window == null ? undefined : range.start.nanoseconds}
                onChange={(start) =>
                  setWindow(new TimeRange(start, range.end).makeValid())
                }
                bound="start"
                resolution={TimeSpan.MINUTE}
                placeholder="Set a start time"
                aria-label="From"
              />
            </Input.Item>
            <Input.Item x label="To" className={CSS.BE("range-filter", "bound")}>
              <Input.DateTime
                value={window == null ? undefined : range.end.nanoseconds}
                onChange={(end) =>
                  setWindow(new TimeRange(range.start, end).makeValid())
                }
                bound="end"
                resolution={TimeSpan.MINUTE}
                placeholder="Set an end time"
                aria-label="To"
              />
            </Input.Item>
          </Flex.Box>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

const ParentItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<ranger.Key>): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Range>(itemKey);
  if (item == null) return null;
  return (
    <Select.Item<ranger.Key> itemKey={itemKey} {...rest}>
      <Ranger.StageIcon timeRange={item.timeRange} />
      {item.name}
    </Select.Item>
  );
};

const parentItem = Component.renderProp(ParentItem);

const ParentPicker = ({
  trigger,
  location,
  surfaceID,
  className,
  ...frame
}: PickerProps): ReactElement => {
  const [conditions, setConditions] = useConditions();
  const value = find(conditions, "parent")?.keys ?? [];
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client) => client.ranges,
    { name: "ranges" },
  );
  const handleChange = (keys: ranger.Key[]): void =>
    setConditions(
      keys.length === 0
        ? drop(conditions, "parent")
        : put(conditions, { property: "parent", keys }),
    );
  return (
    <Dialog.Frame variant="floating" location={location} {...frame}>
      <Select.Frame<ranger.Key, ranger.Range | undefined>
        multiple
        value={value}
        onChange={handleChange}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={more}
      >
        {trigger}
        <Select.Dialog
          surface="menu"
          data-surface={surfaceID}
          className={pickerClass(className, true)}
        >
          <Select.Body<ranger.Key>
            resourceName="range"
            onSearch={search}
            status={status}
          >
            {parentItem}
          </Select.Body>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

interface PropertyInfo {
  name: string;
  icon: Icon.FC;
}

const PROPERTIES: Record<Property, PropertyInfo> = {
  labels: { name: "Labels", icon: Icon.Label },
  stage: { name: "Stage", icon: Icon.ToDo },
  time: { name: "Time", icon: Icon.Time },
  parent: { name: "Parent", icon: Icon.Range },
  other: { name: "Filter", icon: Icon.Filter },
};

const PICKERS: Record<MenuProperty, (props: PickerProps) => ReactElement> = {
  labels: LabelPicker,
  stage: StagePicker,
  time: TimePicker,
  parent: ParentPicker,
};

const MENU_PROPERTIES: MenuProperty[] = ["labels", "stage", "time", "parent"];

interface PropertyTriggerProps {
  property: MenuProperty;
}

/** A property of the filter menu. It opens its values beside the menu. */
const PropertyTrigger = ({ property }: PropertyTriggerProps): ReactElement => {
  const { triggerRef, visible } = Dialog.useContext();
  const { name, icon: PropertyIcon } = PROPERTIES[property];
  return (
    <Button.Button
      ref={triggerRef}
      variant="text"
      gap="small"
      justify="between"
      full="x"
      aria-haspopup="dialog"
      aria-expanded={visible}
      data-property={property}
      className={CSS.BE("range-filter", "property")}
    >
      <Flex.Box x align="center" gap="small" el="span">
        <PropertyIcon />
        {name}
      </Flex.Box>
      <Text.Text el="span" color={8}>
        <Icon.Caret.Right />
      </Text.Text>
    </Button.Button>
  );
};

/** The properties of the filter menu. Each opens its values beside the menu. */
export const MenuItems = (): ReactElement => {
  const { aiming, surface, frame, picker } = useSubmenus();
  return (
    <Flex.Box
      y
      empty
      className={CSS.cls(
        CSS.BE("range-filter", "menu"),
        aiming && CSS.BEM("range-filter", "menu", "aiming"),
      )}
      {...surface}
    >
      {MENU_PROPERTIES.map((property) => {
        const Picker = PICKERS[property];
        return (
          <Picker
            key={property}
            location={BESIDE}
            {...frame(property)}
            {...picker(property)}
            trigger={<PropertyTrigger property={property} />}
          />
        );
      })}
    </Flex.Box>
  );
};

/** @returns "is" for one value, else "is any of". */
const isAnyOf = (count: number): string => (count === 1 ? "is" : "is any of");

/** @returns the names, or a count when there are more than two. */
const summarize = (names: string[], noun: string): string =>
  names.length > 2 ? `${names.length} ${noun}` : names.join(", ");

type KeyMatch = Exclude<LabelMatch, "empty">;

const LABEL_MATCHES: Record<KeyMatch, string> = {
  any: "has any of",
  all: "has all of",
  none: "has none of",
};

const KEY_MATCHES = Object.keys(LABEL_MATCHES) as KeyMatch[];

interface ChipValueProps<P extends Property> {
  condition: ConditionOf<P>;
}

interface MatchSelectProps {
  condition: ConditionOf<"labels"> & { match: KeyMatch };
}

const MatchSelect = ({ condition }: MatchSelectProps): ReactElement => {
  const [conditions, setConditions] = useConditions();
  return (
    <Dialog.Frame variant="floating" location={BELOW}>
      <Select.Frame<KeyMatch, undefined>
        value={condition.match}
        onChange={(match: KeyMatch) =>
          setConditions(put(conditions, { ...condition, match }))
        }
        closeDialogOnSelect
      >
        <View.FilterChipTrigger textColor={9}>
          {LABEL_MATCHES[condition.match]}
        </View.FilterChipTrigger>
        <Select.Dialog surface="menu" className={pickerClass()}>
          <Select.List>
            {KEY_MATCHES.map((key) => (
              <Select.Item<KeyMatch> key={key} itemKey={key}>
                {LABEL_MATCHES[key]}
              </Select.Item>
            ))}
          </Select.List>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

const LabelsValue = ({ condition }: ChipValueProps<"labels">): ReactElement => {
  const { data: labels = [] } = Label.useResultMultiple({ keys: condition.keys });
  const { match } = condition;
  return (
    <>
      {match === "empty" ? (
        <View.FilterChipOperator>is</View.FilterChipOperator>
      ) : (
        <MatchSelect condition={{ ...condition, match }} />
      )}
      <LabelPicker
        location={BELOW}
        trigger={
          <View.FilterChipTrigger>
            {match === "empty" ? (
              "No labels"
            ) : (
              <>
                <span className={CSS.BE("range-filter", "dots")}>
                  {labels.map(({ key, color }) => (
                    <Icon.Circle key={key} color={color} size="2.5em" />
                  ))}
                </span>
                {summarize(
                  labels.map(({ name }) => name),
                  "labels",
                )}
              </>
            )}
          </View.FilterChipTrigger>
        }
      />
    </>
  );
};

const StageValue = ({ condition }: ChipValueProps<"stage">): ReactElement => (
  <>
    <View.FilterChipOperator>
      {isAnyOf(condition.stages.length)}
    </View.FilterChipOperator>
    <StagePicker
      location={BELOW}
      trigger={
        <View.FilterChipTrigger>
          {condition.stages.map((stage) => {
            const StageIcon = Ranger.STAGE_ICONS[stage];
            return <StageIcon key={stage} />;
          })}
          {summarize(
            condition.stages.map((stage) => Ranger.STAGE_NAMES[stage]),
            "stages",
          )}
        </View.FilterChipTrigger>
      }
    />
  </>
);

const formatWindow = (window: TimeSpan | TimeRange): string => {
  if (window instanceof TimeSpan)
    return findPreset(window)?.name.toLowerCase() ?? `the last ${window.toString()}`;
  return window.toPrettyString();
};

const TimeValue = ({ condition }: ChipValueProps<"time">): ReactElement => (
  <>
    <View.FilterChipOperator>overlaps</View.FilterChipOperator>
    <TimePicker
      location={BELOW}
      trigger={
        <View.FilterChipTrigger>
          {formatWindow(condition.window)}
        </View.FilterChipTrigger>
      }
    />
  </>
);

const ParentValue = ({ condition }: ChipValueProps<"parent">): ReactElement => {
  const { data: ranges = [] } = Ranger.useResultMultiple({ keys: condition.keys });
  return (
    <>
      <View.FilterChipOperator>
        {isAnyOf(condition.keys.length)}
      </View.FilterChipOperator>
      <ParentPicker
        location={BELOW}
        trigger={
          <View.FilterChipTrigger>
            {summarize(
              ranges.map(({ name }) => name),
              "ranges",
            )}
          </View.FilterChipTrigger>
        }
      />
    </>
  );
};

const ChipValue = ({ condition }: ChipValueProps<Property>): ReactElement => {
  switch (condition.property) {
    case "labels":
      return <LabelsValue condition={condition} />;
    case "stage":
      return <StageValue condition={condition} />;
    case "time":
      return <TimeValue condition={condition} />;
    case "parent":
      return <ParentValue condition={condition} />;
    case "other":
      return (
        <View.FilterChipValue>{xfilter.describe(condition.where)}</View.FilterChipValue>
      );
  }
};

interface ChipProps {
  condition: Condition;
  editable: boolean;
  onRemove: () => void;
}

const Chip = ({ condition, editable, onRemove }: ChipProps): ReactElement => {
  const { name, icon } = PROPERTIES[condition.property];
  return (
    <View.FilterChip>
      <View.FilterChipName icon={icon}>{name}</View.FilterChipName>
      <ChipValue condition={condition} />
      {editable && <View.FilterChipRemove name={name} onClick={onRemove} />}
    </View.FilterChip>
  );
};

/** One chip for each condition of the view's filter. Empty, it is absent. */
export const Chips = (): ReactElement | null => {
  const { editable } = View.useContext();
  const [conditions, setConditions] = useConditions();
  if (conditions.length === 0) return null;
  return (
    <>
      {conditions.map((condition, i) => (
        <Chip
          key={`${condition.property}-${i}`}
          condition={condition}
          editable={editable}
          onRemove={() => setConditions(conditions.filter((c) => c !== condition))}
        />
      ))}
      {editable && conditions.length > 1 && (
        <Button.Button
          variant="text"
          size="small"
          textColor={9}
          onClick={() => setConditions([])}
        >
          Clear
        </Button.Button>
      )}
    </>
  );
};

/** @returns whether the filter has no conditions. */
export const isEmpty = (where?: ranger.Where): boolean => decode(where).length === 0;
