// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/modbus/device/ByteOrder.css";

import { type CSSProperties, type ReactElement } from "react";

import { Option } from "@/feature/modbus/device/Option";
import { type ByteOrder, swapsOf } from "@/feature/modbus/device/types";
import { CSS } from "@/platform/css";

const ORDER_NAMES: Record<ByteOrder, string> = {
  ABCD: "Big-endian",
  BADC: "Byte swap",
  CDAB: "Word swap",
  DCBA: "Little-endian",
};

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

export interface ByteOrderOptionProps {
  order: ByteOrder;
  /** The size of the value in bytes. */
  size: number;
  /** Stands for the device's order, so the option names the order, not its bytes. */
  device?: boolean;
}

/** A byte order as a menu option: its bytes in the order the device sends them. */
export const ByteOrderOption = ({
  order,
  size,
  device = false,
}: ByteOrderOptionProps): ReactElement => {
  const wire = wireOrder(size, order);
  return (
    <Option
      glyph={<Glyph order={wire} device={device} />}
      name={device ? "From device" : ORDER_NAMES[order]}
      description={device ? ORDER_NAMES[order] : wire.join("")}
    />
  );
};
