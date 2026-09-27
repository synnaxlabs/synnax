// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";

import { type SparkplugDataType } from "@/feature/mqtt/task/types";

const renderSelect = Component.renderProp(
  (
    p: Omit<Select.SingleSimpleProps<SparkplugDataType>, "children" | "resourceName">,
  ) => (
    <Select.Simple<SparkplugDataType>
      {...p}
      resourceName="Sparkplug B type"
      location="bottom"
    >
      <Select.Item itemKey="int8">Int8</Select.Item>
      <Select.Item itemKey="int16">Int16</Select.Item>
      <Select.Item itemKey="int32">Int32</Select.Item>
      <Select.Item itemKey="int64">Int64</Select.Item>
      <Select.Item itemKey="uint8">UInt8</Select.Item>
      <Select.Item itemKey="uint16">UInt16</Select.Item>
      <Select.Item itemKey="uint32">UInt32</Select.Item>
      <Select.Item itemKey="uint64">UInt64</Select.Item>
      <Select.Item itemKey="float">Float</Select.Item>
      <Select.Item itemKey="double">Double</Select.Item>
      <Select.Item itemKey="boolean">Boolean</Select.Item>
      <Select.Item itemKey="string">String</Select.Item>
      <Select.Item itemKey="date_time">DateTime</Select.Item>
    </Select.Simple>
  ),
);

export interface SparkplugTypeFieldProps extends Omit<
  PForm.FieldProps<SparkplugDataType>,
  "children"
> {}

export const SparkplugTypeField = (props: SparkplugTypeFieldProps) => (
  <PForm.Field<SparkplugDataType> {...props}>{renderSelect}</PForm.Field>
);
