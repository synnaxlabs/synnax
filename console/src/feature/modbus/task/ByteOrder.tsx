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
import { DataType, Density } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { useFromConfig } from "@/feature/modbus/device/queries";
import {
  BYTE_ORDERS,
  type ByteOrder,
  byteOrderOf,
  swapsOf,
} from "@/feature/modbus/device/types";
import {
  MENU_PROPS,
  Option,
  SHADOW_TRIGGER_PROPS,
} from "@/feature/modbus/task/Register";
import { CSS } from "@/platform/css";

type Order = "device" | ByteOrder;

const ORDER_NAMES: Record<ByteOrder, string> = {
  ABCD: "Big-endian",
  BADC: "Byte swap",
  CDAB: "Word swap",
  DCBA: "Little-endian",
};

// A word has no word order, so its one swap reverses its bytes.
const WORD_ORDERS: ByteOrder[] = ["ABCD", "DCBA"];

// The bytes in the order the device sends them, as letters with A the most
// significant. The Driver reads the low word first unless the words are swapped.
const wireOrder = (size: number, order: ByteOrder): string[] => {
  const { swapBytes, swapWords } = swapsOf(order);
  const pairs: string[][] = [];
  for (let i = 0; i < Math.max(size, 2); i += 2)
    pairs.push([String.fromCharCode(65 + i), String.fromCharCode(66 + i)]);
  if (!swapWords) pairs.reverse();
  return pairs.flatMap(([a, b]) => (swapBytes ? [b, a] : [a, b]));
};

const orderOf = (
  multiWord: boolean,
  swapBytes: boolean,
  swapWords: boolean,
): ByteOrder =>
  multiWord ? byteOrderOf({ swapBytes, swapWords }) : swapBytes ? "DCBA" : "ABCD";

const GLYPH_WIDTH = 30;

interface GlyphProps {
  order: string[];
  device?: boolean;
}

// The bytes as slices of one fixed width, darkest for the most significant. Each swap
// keeps its shape at any data type size.
const Glyph = ({ order, device = false }: GlyphProps): ReactElement => {
  const width = (GLYPH_WIDTH - (order.length - 1)) / order.length;
  return (
    <svg
      viewBox={`0 0 ${GLYPH_WIDTH} 12`}
      className={CSS.cls(
        CSS.BE("modbus", "order-glyph"),
        device && CSS.BEM("modbus", "order-glyph", "device"),
      )}
      aria-hidden
    >
      {order.map((b, i) => (
        <rect
          key={b}
          x={i * (width + 1)}
          width={width}
          height="12"
          style={
            {
              "--rank": (b.charCodeAt(0) - 65) / Math.max(order.length - 1, 1),
            } as CSSProperties
          }
        />
      ))}
    </svg>
  );
};

export interface ByteOrderFieldProps {
  /** The register channel's form path. */
  path: string;
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
  const bytesPath = `${path}.bytesSwapped`;
  const wordsPath = `${path}.wordsSwapped`;
  const bytesSwapped = Form.useFieldValue<boolean | undefined>(bytesPath, {
    optional: true,
  });
  const wordsSwapped = Form.useFieldValue<boolean | undefined>(wordsPath, {
    optional: true,
  });
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
        <Option
          glyph={<Glyph order={wireOrder(size, device)} device />}
          name="From device"
          description={ORDER_NAMES[device]}
        />
      </Select.Item>
      {(multiWord ? BYTE_ORDERS : WORD_ORDERS).map((order) => {
        const wire = wireOrder(size, order);
        return (
          <Select.Item<Order> key={order} itemKey={order}>
            <Option
              glyph={<Glyph order={wire} />}
              name={ORDER_NAMES[order]}
              description={wire.join("")}
            />
          </Select.Item>
        );
      })}
    </Select.Simple>
  );
};
