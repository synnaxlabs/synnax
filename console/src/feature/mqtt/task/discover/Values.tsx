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
import { Header } from "@synnaxlabs/lyra/header";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Skeleton } from "@synnaxlabs/lyra/skeleton";
import { Text } from "@synnaxlabs/lyra/text";
import { Fragment } from "react";

import { Picker } from "@/feature/mqtt/task/discover/picker";
import { CSS } from "@/platform/css";

/** A value the pane offers: a field of a JSON topic or a Sparkplug B tag. */
export interface Value {
  key: string;
  name: string;
  sample: string | null;
  dataType: string;
  /** The task reads it already. */
  taken: boolean;
  unsupported?: boolean;
}

export interface ValueGroup {
  title?: string;
  values: Value[];
}

interface PaneHeaderProps {
  title: string;
  note?: string;
  free: number;
  picked: number;
  /** Absent for a pane that takes one value. */
  onPickAll?: () => void;
  onClear: () => void;
}

export const PaneHeader = ({
  title,
  note,
  free,
  picked,
  onPickAll,
  onClear,
}: PaneHeaderProps) => (
  <Header.Header bordered={false} className={CSS.BE("mqtt-discover", "pane-header")}>
    <Header.Title weight={500} color={10} variant="code">
      {title}
      {note != null && (
        <Text.Text el="span" level="small" color={9}>
          {note}
        </Text.Text>
      )}
    </Header.Title>
    <Header.Actions>
      {picked > 0 ? (
        <Button.Button variant="text" size="small" onClick={onClear}>
          Clear
        </Button.Button>
      ) : (
        onPickAll != null && (
          <Button.Button
            variant="text"
            size="small"
            disabled={free === 0}
            onClick={onPickAll}
          >
            Select all
          </Button.Button>
        )
      )}
    </Header.Actions>
  </Header.Header>
);

interface HeadProps {
  noun: string;
}

const Head = ({ noun }: HeadProps) => (
  <div className={CSS.BE("mqtt-discover", "values-head")}>
    <span />
    {[noun, "Last value", "Type"].map((label) => (
      <Text.Text key={label} level="small" weight={500} color={9}>
        {label}
      </Text.Text>
    ))}
  </div>
);

const Item = (props: List.ItemRenderProps<string>) => {
  const value = List.useItem<string, Value>(props.itemKey);
  const { selected, onSelect } = Select.useItemState(props.itemKey);
  if (value == null) return null;
  const locked = value.taken || value.unsupported === true;
  let dataType = value.dataType;
  if (value.taken) dataType = "In task";
  else if (value.unsupported === true) dataType = "Not supported";
  return (
    <Select.Item<string>
      {...props}
      disabled={locked}
      className={CSS.BE("mqtt-discover", "value")}
    >
      <Input.Checkbox
        value={value.taken || selected}
        onChange={onSelect}
        disabled={locked}
        size="small"
        aria-label={`Record ${value.name}`}
      />
      <Text.Text level="small" weight={500} color={locked ? 9 : 10} overflow="fade">
        {value.name}
      </Text.Text>
      <Text.Text level="small" color={9} overflow="fade" variant="code">
        {value.sample ?? "–"}
      </Text.Text>
      <Text.Text level="small" color={9} overflow="fade">
        {dataType}
      </Text.Text>
    </Select.Item>
  );
};

const item = Component.renderProp(Item);

interface GroupListProps extends Pick<ValuesProps, "value" | "onChange"> {
  values: Value[];
}

const GroupList = ({ values, value, onChange }: GroupListProps) => {
  const { data, getItem } = List.useStaticData<string, Value>({ data: values });
  return (
    <Select.Frame<string, Value>
      multiple
      data={data}
      getItem={getItem}
      value={value}
      onChange={onChange}
    >
      <List.Scroll className={CSS.BE("mqtt-discover", "value-list")}>
        <List.Items<string, Value>>{item}</List.Items>
      </List.Scroll>
    </Select.Frame>
  );
};

export interface ValuesProps {
  noun: string;
  groups: ValueGroup[];
  value: string[];
  onChange: (value: string[]) => void;
}

export const Values = ({ noun, groups, value, onChange }: ValuesProps) => (
  <div className={CSS.BE("mqtt-discover", "values")}>
    <Head noun={noun} />
    {groups.map(({ title, values }) => (
      <Fragment key={title ?? ""}>
        {title != null && (
          <Text.Text
            level="small"
            weight={500}
            color={9}
            className={CSS.BE("mqtt-discover", "section")}
          >
            {title}
          </Text.Text>
        )}
        <GroupList values={values} value={value} onChange={onChange} />
      </Fragment>
    ))}
  </div>
);

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
    <Header.Header bordered={false} className={CSS.BE("mqtt-discover", "pane-header")}>
      {title == null ? (
        <Header.Title weight={500}>
          <Skeleton.Skeleton style={{ width: "24rem" }} />
        </Header.Title>
      ) : (
        <Header.Title weight={500} color={10} variant="code">
          {title}
        </Header.Title>
      )}
    </Header.Header>
    <div aria-hidden className={CSS.BE("mqtt-discover", "values")}>
      <Head noun={noun} />
      <div className={CSS.BE("mqtt-discover", "value-list")}>
        {SKELETON_WIDTHS.map((widths, i) => (
          <div key={i} className={CSS.BE("mqtt-discover", "value-skeleton")}>
            <Picker.CheckSkeleton />
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
