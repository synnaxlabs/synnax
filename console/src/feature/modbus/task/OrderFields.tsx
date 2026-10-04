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
import { DataType, Density } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useFromConfig } from "@/feature/modbus/device/queries";
import { CSS } from "@/platform/css";

type Order = "device" | "swapped" | "not_swapped";

const LABELS = { bytesSwapped: "Bytes", wordsSwapped: "Words" } as const;

interface SelectOrderProps {
  path: string;
  field: keyof typeof LABELS;
  deviceSwapped?: boolean;
}

const describe = (swapped: boolean): string => (swapped ? "swapped" : "not swapped");

const SelectOrder = ({
  path,
  field,
  deviceSwapped,
}: SelectOrderProps): ReactElement => {
  const label = LABELS[field];
  const fieldPath = `${path}.${field}`;
  const ctx = Form.useContext();
  const value = Form.useFieldValue<boolean | undefined>(fieldPath, { optional: true });
  const order: Order = value == null ? "device" : value ? "swapped" : "not_swapped";
  const deviceOption =
    deviceSwapped == null ? "device" : `device (${describe(deviceSwapped)})`;
  return (
    <Select.Simple<Order>
      value={order}
      preview={ctx.mode === "preview"}
      onChange={(next: Order) =>
        ctx.set(fieldPath, next === "device" ? undefined : next === "swapped")
      }
      allowNone={false}
      resourceName={`${label.toLowerCase()} order`}
    >
      <Select.Item<Order> itemKey="device">{`${label}: ${deviceOption}`}</Select.Item>
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
export const OrderFields = ({ path, dataType }: OrderFieldsProps): ReactElement => {
  const connection = useFromConfig()?.properties.connection;
  return (
    <Flex.Box x pack className={CSS.B("modbus-order")}>
      <SelectOrder
        path={path}
        field="bytesSwapped"
        deviceSwapped={connection?.swapBytes}
      />
      {new DataType(dataType).density.valueOf() > Density.BIT16.valueOf() && (
        <SelectOrder
          path={path}
          field="wordsSwapped"
          deviceSwapped={connection?.swapWords}
        />
      )}
    </Flex.Box>
  );
};
