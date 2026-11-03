// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { DataType, deep, Density } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { ByteOrderOption } from "@/feature/modbus/device/ByteOrder";
import { MENU_PROPS, SHADOW_TRIGGER_PROPS } from "@/feature/modbus/device/Option";
import { useFromConfig } from "@/feature/modbus/device/queries";
import {
  BYTE_ORDERS,
  type ByteOrder,
  byteOrderOf,
  swapsOf,
} from "@/feature/modbus/device/types";

type Order = "device" | ByteOrder;

// A word has no word order, so its one swap reverses its bytes.
const WORD_ORDERS: ByteOrder[] = ["ABCD", "DCBA"];

const orderOf = (
  multiWord: boolean,
  swapBytes: boolean,
  swapWords: boolean,
): ByteOrder =>
  multiWord ? byteOrderOf({ swapBytes, swapWords }) : swapBytes ? "DCBA" : "ABCD";

interface Swapped {
  bytesSwapped?: boolean;
  wordsSwapped?: boolean;
}

export interface ByteOrderFieldProps {
  /** The register channel's form path. */
  path: deep.Target<Swapped>;
  /** The channel's data type. */
  dataType: string;
}

/**
 * Selects the order of a register channel's bytes and words: the device's order, or a
 * swap of it. A data type that fits in one word has no word order, so it offers only
 * the two byte orders.
 */
export const ByteOrderField = ({
  path,
  dataType,
}: ByteOrderFieldProps): ReactElement => {
  const connection = useFromConfig()?.properties.connection;
  const ctx = Form.useContext();
  const { bytesSwapped: bytesPath, wordsSwapped: wordsPath } = deep.widen(path);
  const bytesSwapped = Form.useFieldValue(bytesPath, { optional: true });
  const wordsSwapped = Form.useFieldValue(wordsPath, { optional: true });
  const size = new DataType(dataType).density.valueOf();
  const multiWord = size > Density.BIT16.valueOf();
  const deviceBytes = connection?.swapBytes ?? false;
  const deviceWords = connection?.swapWords ?? false;
  const device = orderOf(multiWord, deviceBytes, deviceWords);
  const value: Order =
    bytesSwapped == null && wordsSwapped == null
      ? "device"
      : orderOf(multiWord, bytesSwapped ?? deviceBytes, wordsSwapped ?? deviceWords);
  const handleChange = (next: Order) => {
    if (next === "device") {
      ctx.set(bytesPath, undefined);
      ctx.set(wordsPath, undefined);
      return;
    }
    const { swapBytes, swapWords } = swapsOf(next);
    ctx.set(bytesPath, swapBytes);
    ctx.set(wordsPath, multiWord ? swapWords : undefined);
  };
  return (
    <Select.Simple<Order>
      value={value}
      onChange={handleChange}
      preview={ctx.mode === "preview"}
      allowNone={false}
      resourceName="byte order"
      triggerProps={SHADOW_TRIGGER_PROPS}
      dialogProps={MENU_PROPS}
    >
      <Select.Item<Order> itemKey="device">
        <ByteOrderOption order={device} size={size} device />
      </Select.Item>
      {(multiWord ? BYTE_ORDERS : WORD_ORDERS).map((order) => (
        <Select.Item<Order> key={order} itemKey={order}>
          <ByteOrderOption order={order} size={size} />
        </Select.Item>
      ))}
    </Select.Simple>
  );
};
