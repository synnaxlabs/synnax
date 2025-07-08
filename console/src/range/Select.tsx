// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Align,
  Button,
  Component,
  Dialog,
  Icon,
  Input,
  List,
  Ranger,
  Select,
  Status,
  Tag,
  Text,
  TimeSpan,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/range/Create";
import { useSelect, useSelectKeys } from "@/range/selectors";
import { type Range } from "@/range/slice";

interface SelectMultipleRangesProps extends Select.MultipleProps<string> {}

const dynamicIcon = (
  <Icon.Dynamic style={{ color: "var(--pluto-error-p1)", filter: "opacity(0.8)" }} />
);

const listItem = Component.renderProp((props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const range = List.useItem<string, Range>(itemKey);
  if (range == null) return null;
  const { variant, name } = range;
  return (
    <List.Item {...props}>
      <Text.Text level="p">{name}</Text.Text>
      {variant === "dynamic" ? (
        <Text.Text level="p" shade={11}>
          {new TimeSpan(range.span).toString()}
        </Text.Text>
      ) : (
        <Ranger.TimeRangeChip level="small" timeRange={range.timeRange} />
      )}
    </List.Item>
  );
});

interface RenderTagProps {
  itemKey: string;
  onClose: (itemKey: string) => void;
}

const RangeTag = ({ itemKey, onClose }: RenderTagProps): ReactElement | null => {
  const range = List.useItem<string, Range>(itemKey);
  if (range == null) return null;
  return (
    <Tag.Tag
      icon={range?.variant === "dynamic" ? dynamicIcon : <Icon.Range />}
      onClose={() => onClose(itemKey)}
      shade={11}
      level="small"
    >
      {range.name}
    </Tag.Tag>
  );
};

const SelectMultipleRanges = ({
  value,
  onChange,
}: SelectMultipleRangesProps): ReactElement => {
  const data = useSelectKeys();
  const { onSelect, ...selectProps } = Select.useMultiple({ data, value, onChange });
  return (
    <Select.Dialog data={data} useItem={useSelect} onSelect={onSelect} {...selectProps}>
      <Dialog.Trigger>
        {value.map((key) => (
          <RangeTag key={key} itemKey={key} onClose={onSelect} />
        ))}
      </Dialog.Trigger>
      <Dialog.Content>
        <List.Items emptyContent={<SelectEmptyContent />}>{listItem}</List.Items>
      </Dialog.Content>
    </Select.Dialog>
  );
};

interface SelectSingleRangeProps extends Select.SingleProps<string> {}

const SelectRange = ({ value, onChange }: SelectSingleRangeProps): ReactElement => {
  const data = useSelectKeys();
  const { onSelect, ...selectProps } = Select.useSingle({ data, value, onChange });
  const item = useSelect(value);
  return (
    <Select.Dialog data={data} useItem={useSelect} onSelect={onSelect} {...selectProps}>
      <Dialog.Trigger>
        {item != null ? <RangeTag itemKey={item.key} onClose={onSelect} /> : null}
      </Dialog.Trigger>
      <Dialog.Content>
        <List.Items emptyContent={<SelectEmptyContent />}>{listItem}</List.Items>
      </Dialog.Content>
    </Select.Dialog>
  );
};

interface SelectMultipleInputItemProps
  extends Omit<Input.ItemProps, "label" | "onChange">,
    SelectMultipleRangesProps {
  value: string[];
  onChange: (value: string[]) => void;
  selectProps?: Partial<SelectMultipleRangesProps>;
}

const SelectEmptyContent = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Align.Center style={{ height: 150 }} x>
      <Status.Text variant="disabled" hideIcon>
        No Ranges:
      </Status.Text>
      <Button.Button variant="outlined" onClick={() => placeLayout(CREATE_LAYOUT)}>
        Define a Range
      </Button.Button>
    </Align.Center>
  );
};

export const SelectMultipleInputItem = ({
  value,
  onChange,
  selectProps,
  ...rest
}: SelectMultipleInputItemProps): ReactElement => (
  <Input.Item x label="Ranges" {...rest}>
    <SelectMultipleRanges value={value} onChange={onChange} {...selectProps} />
  </Input.Item>
);

interface SelectInputItemProps
  extends Omit<Input.ItemProps, "label" | "onChange">,
    SelectSingleRangeProps {
  selectProps?: Partial<SelectSingleRangeProps>;
}

export const SelectInputItem = ({
  value,
  onChange,
  selectProps,
  ...rest
}: SelectInputItemProps): ReactElement => (
  <Input.Item label="Range:" {...rest}>
    <SelectRange value={value} onChange={onChange} {...selectProps} />
  </Input.Item>
);
