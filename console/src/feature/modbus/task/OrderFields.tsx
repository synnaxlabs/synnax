// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { DataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useFromConfig } from "@/feature/modbus/device/queries";
import { CSS } from "@/platform/css";

type Order = "device" | "swapped" | "not_swapped";

const SUBJECTS = {
  bytesSwapped: { label: "Bytes", connection: "swapBytes" },
  wordsSwapped: { label: "Words", connection: "swapWords" },
} as const;

interface SelectOrderProps {
  path: string;
  field: keyof typeof SUBJECTS;
}

const describe = (swapped: boolean): string => (swapped ? "swapped" : "not swapped");

const SelectOrder = ({ path, field }: SelectOrderProps): ReactElement => {
  const { label, connection } = SUBJECTS[field];
  const fieldPath = `${path}.${field}`;
  const ctx = Form.useContext();
  const value = Form.useFieldValue<boolean | undefined>(fieldPath, { optional: true });
  const deviceSwapped = useFromConfig()?.properties.connection[connection];
  const order: Order = value == null ? "device" : value ? "swapped" : "not_swapped";
  const deviceName =
    deviceSwapped == null ? "device" : `device (${describe(deviceSwapped)})`;
  return (
    <Select.Simple<Order>
      value={order}
      onChange={(next: Order) =>
        ctx.set(fieldPath, next === "device" ? undefined : next === "swapped")
      }
      allowNone={false}
      resourceName={`${label.toLowerCase()} order`}
    >
      <Select.Item<Order> itemKey="device">{`${label}: ${deviceName}`}</Select.Item>
      <Select.Item<Order> itemKey="swapped">{`${label}: ${describe(true)}`}</Select.Item>
      <Select.Item<Order> itemKey="not_swapped">
        {`${label}: ${describe(false)}`}
      </Select.Item>
    </Select.Simple>
  );
};

export interface OrderFieldsProps {
  /** The path of the register channel in the form. */
  path: string;
  /** The channel's data type. */
  dataType: string;
}

/**
 * Selects a register channel's byte and word order: the device's order, or an
 * override. The device option shows the order the device currently sets. A data type
 * that fits in one register has no word order, so it shows the byte order alone.
 */
export const OrderFields = ({ path, dataType }: OrderFieldsProps): ReactElement => (
  <Flex.Box x pack className={CSS.BE("modbus", "order")}>
    <SelectOrder path={path} field="bytesSwapped" />
    {new DataType(dataType).density.valueOf() > 2 && (
      <SelectOrder path={path} field="wordsSwapped" />
    )}
  </Flex.Box>
);
