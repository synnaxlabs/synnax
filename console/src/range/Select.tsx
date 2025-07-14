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

interface SelectMultipleRangesProps extends Select.MultipleProps<string, Range> {}

const dynamicIcon = (
  <Icon.Dynamic style={{ color: "var(--pluto-error-p1)", filter: "opacity(0.8)" }} />
);

const listItem = Component.renderProp((props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const range = useSelect(itemKey);
  const { selected, onSelect } = Select.useItemState(itemKey);
  if (range == null) return null;
  const { variant, name } = range;
  return (
    <List.Item {...props} selected={selected} onSelect={onSelect}>
      <Text.Text level="p" style={{ width: 100 }}>
        {name}
      </Text.Text>
      {variant === "dynamic" ? (
        <Text.WithIcon level="p" shade={11} startIcon={dynamicIcon}>
          {new TimeSpan(range.span).toString()}
        </Text.WithIcon>
      ) : (
        <Ranger.TimeRangeChip level="small" timeRange={range.timeRange} />
      )}
    </List.Item>
  );
});

interface RenderTagProps {
  itemKey: string;
}

const RangeTag = ({ itemKey }: RenderTagProps): ReactElement | null => {
  const range = useSelect(itemKey);
  const { onSelect } = Select.useItemState(itemKey);
  if (range == null) return null;
  return (
    <Tag.Tag
      icon={range?.variant === "dynamic" ? dynamicIcon : <Icon.Range />}
      onClose={onSelect}
      shade={11}
      level="small"
      size="small"
    >
      {range.name}
    </Tag.Tag>
  );
};

const SelectMultipleRanges = ({
  value,
  onChange,
  ...rest
}: SelectMultipleRangesProps): ReactElement => {
  const data = useSelectKeys();
  return (
    <Dialog.Frame variant="connected" {...rest}>
      <Select.Frame multiple data={data} onChange={onChange} value={value}>
        <Select.MultipleTrigger>
          {(props) => <RangeTag {...props} />}
        </Select.MultipleTrigger>
        <Select.Dialog<string>
          searchPlaceholder="Search Ranges..."
          emptyContent={<SelectEmptyContent />}
        >
          {listItem}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

interface SelectSingleRangeProps extends Select.SingleProps<string, Range> {}

const SelectRange = ({ value, onChange }: SelectSingleRangeProps): ReactElement => {
  const data = useSelectKeys();
  return (
    <Dialog.Frame>
      <Select.Frame data={data} onChange={onChange} value={value}>
        <Dialog.Trigger>
          {value != null ? <RangeTag itemKey={value} /> : null}
        </Dialog.Trigger>
        <Select.Dialog<string>
          searchPlaceholder="Search Ranges..."
          emptyContent={<SelectEmptyContent />}
        >
          {listItem}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
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
