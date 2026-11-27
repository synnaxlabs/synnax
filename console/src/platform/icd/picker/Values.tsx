// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { context } from "@synnaxlabs/lyra/context";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Skeleton } from "@synnaxlabs/lyra/skeleton";
import { Text } from "@synnaxlabs/lyra/text";
import { Fragment, type SyntheticEvent } from "react";

import { CSS } from "@/platform/css";
import { useHint } from "@/platform/icd/picker/hint";
import { CheckSkeleton } from "@/platform/icd/picker/Picker";

/** Fixes the checkbox of a value that cannot be picked. */
export interface Lock {
  checked: boolean;
  /** Says why, in place of the type of the value. */
  label: string;
}

/** The lock of a value that the task reads already. */
export const TAKEN: Lock = { checked: true, label: "In task" };

/** The lock of the timestamp that indexes the picked values. */
export const INDEX: Lock = { checked: true, label: "Index" };

/** The lock of a value of a shape that no held message is sure to get. */
export const SPLIT: Lock = { checked: false, label: "Split by value" };

/** @returns the note of a shape whose messages differ only in the value at pointer. */
export const splitNote = (pointer: string): string =>
  `Split by the value at ${pointer}. Add fields in the ICD editor.`;

/** A value the pane offers, such as one field of a message. */
export interface Value {
  key: string;
  name: string;
  sample: string | null;
  dataType: string;
  lock?: Lock;
  /** Shows under the name, such as what a pick of the value changes. */
  note?: string;
  /** The text of the Default input of the value. Absent, the value takes no default. */
  default?: string;
  /** Says why default spells no default of the value. */
  problem?: string;
}

/** Called with the key of a value and the new text of its Default input. */
type DefaultChange = (key: string, text: string) => void;

const [DefaultContext, useDefaultContext] = context.create<DefaultChange | null>({
  defaultValue: null,
  displayName: "Picker.DefaultContext",
});

export interface ValueGroup {
  title?: string;
  /** Shows at the end of the title, such as what a pick uses the group for. */
  note?: string;
  /** Fades the values, such as ones that a pick does not use. */
  dimmed?: boolean;
  values: Value[];
}

/** The values that Select all picks, and how many of them are picked. */
interface PickAllProps {
  free: number;
  picked: number;
  onPickAll: () => void;
  onClear: () => void;
}

/** Without the props of Select all, the header has neither Select all nor Clear. */
type PaneHeaderProps = { title: string; note?: string } & (
  PickAllProps | { [K in keyof PickAllProps]?: never }
);

const Hint = () => (
  <Text.Text level="small" color={9} className={CSS.BE("icd-picker", "hint")}>
    {useHint("Picker.Hint")}
  </Text.Text>
);

const PickAll = ({ free, picked, onPickAll, onClear }: PickAllProps) =>
  picked > 0 ? (
    <Button.Button variant="text" size="small" onClick={onClear}>
      Clear
    </Button.Button>
  ) : (
    <Button.Button
      variant="text"
      size="small"
      disabled={free === 0}
      onClick={onPickAll}
    >
      Select all
    </Button.Button>
  );

/** The head of a pane: its title, Select all or Clear, then the hint of the dialog. */
export const PaneHeader = (props: PaneHeaderProps) => (
  <>
    <Header.Header bordered={false} className={CSS.BE("icd-picker", "pane-header")}>
      <Header.Title weight={500} color={10}>
        {props.title}
        {props.note != null && (
          <Text.Text el="span" level="small" color={9}>
            {props.note}
          </Text.Text>
        )}
      </Header.Title>
      {props.onPickAll != null && (
        <Header.Actions>
          <PickAll {...props} />
        </Header.Actions>
      )}
    </Header.Header>
    <Hint />
  </>
);

interface HeadProps {
  noun: string;
  /** Adds a Default column. */
  defaults?: boolean;
}

const Head = ({ noun, defaults = false }: HeadProps) => (
  <div className={CSS.BE("icd-picker", "values-head")}>
    <span />
    {[noun, "Last value", "Type", ...(defaults ? ["Default"] : [])].map((label) => (
      <Text.Text key={label} level="small" weight={500} color={9}>
        {label}
      </Text.Text>
    ))}
  </div>
);

const stopPropagation = (e: SyntheticEvent) => e.stopPropagation();

interface DefaultInputProps {
  value: Value;
  onChange: DefaultChange;
}

/** The Default input of a value. Clicks and keys in it do not pick the value. */
const DefaultInput = ({
  value: { key, name, default: text = "", problem },
  onChange,
}: DefaultInputProps) => (
  <div onClick={stopPropagation} onKeyDown={stopPropagation}>
    <Input.Text
      size="small"
      value={text}
      onChange={(next) => onChange(key, next)}
      placeholder="None"
      status={problem == null ? undefined : "error"}
      tooltip={problem}
      aria-label={`Default of ${name}`}
    />
  </div>
);

const Cells = ({ value }: { value: Value }) => {
  const { name, sample, dataType, lock, note } = value;
  const onDefaultChange = useDefaultContext();
  return (
    <>
      <Flex.Box y gap="tiny" className={CSS.BE("icd-picker", "value-name")}>
        <Text.Text
          level="small"
          weight={500}
          color={lock != null ? 9 : 10}
          overflow="fade"
        >
          {name}
        </Text.Text>
        {note != null && (
          <Text.Text level="small" color={8} overflow="fade">
            {note}
          </Text.Text>
        )}
      </Flex.Box>
      <Text.Text level="small" color={9} overflow="fade" variant="numeric">
        {sample ?? "–"}
      </Text.Text>
      <Text.Text level="small" color={9} overflow="fade">
        {lock?.label ?? dataType}
      </Text.Text>
      {onDefaultChange != null &&
        (value.default == null ? (
          <span />
        ) : (
          <DefaultInput value={value} onChange={onDefaultChange} />
        ))}
    </>
  );
};

const Item = (props: List.ItemRenderProps<string>) => {
  const value = List.useItem<string, Value>(props.itemKey);
  const { selected, onSelect } = Select.useItemState(props.itemKey);
  if (value == null) return null;
  const { lock } = value;
  const locked = lock != null;
  return (
    <Select.Item<string>
      {...props}
      disabled={locked}
      className={CSS.BE("icd-picker", "value")}
    >
      <Input.Checkbox
        value={lock?.checked ?? selected}
        onChange={onSelect}
        disabled={locked}
        size="small"
        aria-label={`Record ${value.name}`}
      />
      <Cells value={value} />
    </Select.Item>
  );
};

const PreviewItem = (props: List.ItemRenderProps<string>) => {
  const value = List.useItem<string, Value>(props.itemKey);
  if (value == null) return null;
  return (
    <List.Item<string> {...props} className={CSS.BE("icd-picker", "value")}>
      <span />
      <Cells value={value} />
    </List.Item>
  );
};

const item = Component.renderProp(Item);
const previewItem = Component.renderProp(PreviewItem);

interface GroupListProps
  extends
    Pick<ValuesProps, "value" | "onChange">,
    Pick<ValueGroup, "dimmed" | "values"> {}

const GroupList = ({
  values,
  dimmed = false,
  value = [],
  onChange,
}: GroupListProps) => {
  const { data, getItem } = List.useStaticData<string, Value>({ data: values });
  const list = (
    <List.Scroll
      className={CSS.cls(CSS.BE("icd-picker", "value-list"), dimmed && CSS.M("dimmed"))}
    >
      <List.Items<string, Value>>{onChange == null ? previewItem : item}</List.Items>
    </List.Scroll>
  );
  if (onChange == null)
    return (
      <List.Frame<string, Value> data={data} getItem={getItem}>
        {list}
      </List.Frame>
    );
  return (
    <Select.Frame<string, Value>
      multiple
      data={data}
      getItem={getItem}
      value={value}
      onChange={onChange}
    >
      {list}
    </Select.Frame>
  );
};

export interface ValuesProps {
  noun: string;
  groups: ValueGroup[];
  /** The keys of the picked values. */
  value?: string[];
  /** Omitted, the values are a preview with no checkboxes. */
  onChange?: (value: string[]) => void;
  /** Adds a Default column for the values that take a default. */
  onDefaultChange?: DefaultChange;
}

export const Values = ({
  noun,
  groups,
  value,
  onChange,
  onDefaultChange,
}: ValuesProps) => {
  const defaults =
    onDefaultChange != null &&
    groups.some(({ values }) => values.some((v) => v.default != null));
  return (
    <div className={CSS.cls(CSS.BE("icd-picker", "values"), defaults && CSS.M("defaults"))}>
      <Head noun={noun} defaults={defaults} />
      <DefaultContext value={defaults ? onDefaultChange : null}>
        {groups.map(({ title, note, dimmed, values }) => (
          <Fragment key={title ?? ""}>
            {title != null && (
              <Text.Text
                level="small"
                weight={500}
                color={9}
                className={CSS.BE("icd-picker", "section")}
              >
                {title}
                {note != null && (
                  <Text.Text
                    el="span"
                    level="small"
                    className={CSS.BE("icd-picker", "section-note")}
                  >
                    {note}
                  </Text.Text>
                )}
              </Text.Text>
            )}
            <GroupList
              values={values}
              dimmed={dimmed}
              value={value}
              onChange={onChange}
            />
          </Fragment>
        ))}
      </DefaultContext>
    </div>
  );
};

const SKELETON_WIDTHS = [
  ["14ch", "12ch", "6ch"],
  ["11ch", "9ch", "6ch"],
  ["15ch", "12ch", "5ch"],
  ["10ch", "9ch", "6ch"],
];

export interface PaneSkeletonProps extends HeadProps {
  title?: string;
}

export const PaneSkeleton = ({ noun, title }: PaneSkeletonProps) => (
  <>
    <Header.Header bordered={false} className={CSS.BE("icd-picker", "pane-header")}>
      {title == null ? (
        <Header.Title weight={500}>
          <Skeleton.Skeleton style={{ width: "24rem" }} />
        </Header.Title>
      ) : (
        <Header.Title weight={500} color={10}>
          {title}
        </Header.Title>
      )}
    </Header.Header>
    <Hint />
    <div aria-hidden className={CSS.BE("icd-picker", "values")}>
      <Head noun={noun} />
      <div className={CSS.BE("icd-picker", "value-list")}>
        {SKELETON_WIDTHS.map((widths, i) => (
          <div key={i} className={CSS.BE("icd-picker", "value-skeleton")}>
            <CheckSkeleton />
            {widths.map((width, j) => (
              <Text.Text key={j} level="small">
                <Skeleton.Skeleton style={{ width }} />
              </Text.Text>
            ))}
          </div>
        ))}
      </div>
    </div>
  </>
);
