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
  MENU_PROPS,
  Option,
  SHADOW_TRIGGER_PROPS,
} from "@/feature/modbus/task/Register";
import { CSS } from "@/platform/css";

type Swap = "none" | "bytes" | "words" | "both";

type Order = "device" | Swap;

const SWAPS: Record<Swap, [bytes: boolean, words: boolean]> = {
  none: [false, false],
  bytes: [true, false],
  words: [false, true],
  both: [true, true],
};

const SWAP_NAMES: Record<Swap, string> = {
  none: "Big-endian",
  bytes: "Byte swap",
  words: "Word swap",
  both: "Little-endian",
};

const MULTI_WORD_SWAPS: Swap[] = ["none", "bytes", "words", "both"];

// A word has no word order, so its one swap reverses its bytes.
const WORD_SWAPS: Swap[] = ["none", "both"];

// The bytes in the order the device sends them, as letters with A the most
// significant.
const wireOrder = (size: number, [bytes, words]: [boolean, boolean]): string[] => {
  const pairs: string[][] = [];
  for (let i = 0; i < Math.max(size, 2); i += 2)
    pairs.push([String.fromCharCode(65 + i), String.fromCharCode(66 + i)]);
  if (words) pairs.reverse();
  return pairs.flatMap(([a, b]) => (bytes ? [b, a] : [a, b]));
};

const swapOf = (size: number, [bytes, words]: [boolean, boolean]): Swap => {
  if (size <= 2) return bytes ? "both" : "none";
  if (bytes) return words ? "both" : "bytes";
  return words ? "words" : "none";
};

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
  const device: [boolean, boolean] = [
    connection?.swapBytes ?? false,
    multiWord && (connection?.swapWords ?? false),
  ];
  const value: Order =
    bytesSwapped == null && wordsSwapped == null
      ? "device"
      : swapOf(size, [
          bytesSwapped ?? device[0],
          multiWord && (wordsSwapped ?? device[1]),
        ]);
  const handleChange = (next: Order) => {
    if (next === "device") {
      ctx.set(bytesPath, undefined);
      ctx.set(wordsPath, undefined);
      return;
    }
    const [bytes, words] = SWAPS[next];
    ctx.set(bytesPath, bytes);
    ctx.set(wordsPath, multiWord ? words : undefined);
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
          description={SWAP_NAMES[swapOf(size, device)]}
        />
      </Select.Item>
      {(multiWord ? MULTI_WORD_SWAPS : WORD_SWAPS).map((s) => {
        const order = wireOrder(size, SWAPS[s]);
        return (
          <Select.Item<Order> key={s} itemKey={s}>
            <Option
              glyph={<Glyph order={order} />}
              name={SWAP_NAMES[s]}
              description={order.join("")}
            />
          </Select.Item>
        );
      })}
    </Select.Simple>
  );
};
