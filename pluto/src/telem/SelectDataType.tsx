// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/telem/SelectDataType.css";

import { caseconv, DataType } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { Component } from "@/component";
import { CSS } from "@/css";
import { List } from "@/list";
import { Select } from "@/select";
import { resolveDataTypeIcon } from "@/telem/resolveDataTypeIcon";
import { Text } from "@/text";

const ALL_CAPS = new Set([DataType.UUID, DataType.JSON]);

const resolveIcon = (d: DataType) => {
  const Resolved = resolveDataTypeIcon(d);
  return Resolved != null ? <Resolved /> : undefined;
};

const isCode = (d: DataType): boolean =>
  d.isNumeric && d !== DataType.TIMESTAMP && d !== DataType.BOOLEAN;

const resolveName = (d: DataType): string => {
  if (d === DataType.BOOLEAN) return "Boolean";
  if (ALL_CAPS.has(d)) return d.toString().toUpperCase();
  if (isCode(d)) return d.toString();
  return caseconv.capitalize(d.toString());
};

interface Entry extends Select.StaticEntry<string> {
  code: boolean;
}

const DATA: Entry[] = DataType.ALL.filter((d) => d !== DataType.UNKNOWN).map((d) => ({
  key: d.toString(),
  name: resolveName(d),
  icon: resolveIcon(d),
  code: isCode(d),
}));

const FIXED_DENSITY_DATA = DATA.filter((d) => !new DataType(d.key).isVariable);

const listItem = Component.renderProp((props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const item = List.useItem<string, Entry>(itemKey);
  if (item == null) return null;
  const { name, icon, code } = item;
  return (
    <Select.ListItem {...props}>
      {icon}
      {code ? (
        <Text.Text el="span" variant="code">
          {name}
        </Text.Text>
      ) : (
        name
      )}
    </Select.ListItem>
  );
});

export interface SelectDataTypeProps extends Omit<
  Select.StaticProps<string>,
  "data" | "resourceName"
> {
  hideVariableDensity?: boolean;
  hideDataTypes?: DataType[];
}

export const SelectDataType = ({
  hideVariableDensity = false,
  hideDataTypes = DEFAULT_HIDDEN_DATA_TYPES,
  triggerProps,
  value,
  ...rest
}: SelectDataTypeProps): ReactElement => {
  const data = hideVariableDensity ? FIXED_DENSITY_DATA : DATA;
  const filteredData = useMemo(
    () => data.filter((d) => !hideDataTypes.some((h) => h.equals(d.key))),
    [hideDataTypes, data],
  );
  const code = DATA.find((d) => d.key === value)?.code === true;
  return (
    <Select.Static
      {...rest}
      value={value}
      data={filteredData}
      resourceName="data type"
      triggerProps={{
        ...triggerProps,
        className: CSS(
          code && CSS.BEM("select-data-type", "trigger", "code"),
          triggerProps?.className,
        ),
      }}
    >
      {listItem}
    </Select.Static>
  );
};

const DEFAULT_HIDDEN_DATA_TYPES: DataType[] = [];
