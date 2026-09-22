// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Form as PForm, Select } from "@synnaxlabs/pluto";

import { type SparkplugDataType } from "@/feature/mqtt/task/types";

const DATA: Select.StaticEntry<SparkplugDataType>[] = [
  { key: "int8", name: "Int8" },
  { key: "int16", name: "Int16" },
  { key: "int32", name: "Int32" },
  { key: "int64", name: "Int64" },
  { key: "uint8", name: "UInt8" },
  { key: "uint16", name: "UInt16" },
  { key: "uint32", name: "UInt32" },
  { key: "uint64", name: "UInt64" },
  { key: "float", name: "Float" },
  { key: "double", name: "Double" },
  { key: "boolean", name: "Boolean" },
  { key: "string", name: "String" },
  { key: "date_time", name: "DateTime" },
];

const renderSelect = Component.renderProp(
  (
    p: Omit<
      Select.StaticProps<SparkplugDataType, Select.StaticEntry<SparkplugDataType>>,
      "data" | "resourceName"
    >,
  ) => (
    <Select.Static<SparkplugDataType, Select.StaticEntry<SparkplugDataType>>
      {...p}
      data={DATA}
      resourceName="Sparkplug B type"
      location="bottom"
    />
  ),
);

export interface SparkplugTypeFieldProps extends Omit<
  PForm.FieldProps<SparkplugDataType>,
  "children"
> {}

export const SparkplugTypeField = (props: SparkplugTypeFieldProps) => (
  <PForm.Field<SparkplugDataType> {...props}>{renderSelect}</PForm.Field>
);
