// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bus, type library } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { formatEscaped, parseEscaped } from "@/feature/bus/escape";
import { Task } from "@/platform/task";

const PATH = "config.framing";

const TYPE_NAMES = {
  delimiter: "Delimiter",
  fixed: "Fixed length",
  sync: "Sync and length",
  cobs: "COBS",
  slip: "SLIP",
} as const satisfies Record<bus.FramingType, string>;

const BYTE_ORDER_NAMES = {
  little_endian: "Little endian",
  big_endian: "Big endian",
} as const satisfies Record<library.ByteOrder, string>;

const CHECKSUM_NAMES = {
  none: "None",
  crc16_ccitt: "CRC-16/CCITT",
  crc16_modbus: "CRC-16/MODBUS",
  crc32: "CRC-32",
} as const satisfies Record<bus.Checksum, string>;

const LENGTH_SIZES = [1, 2, 4] as const;
type LengthSize = (typeof LENGTH_SIZES)[number];

const TypeField = Form.buildSelectField<bus.FramingType>({
  fieldProps: {
    label: "Framing",
    // A new type starts from its own defaults, so no field of the old type lingers.
    onChange: (type, { set }) => set(PATH, bus.FRAMING_SCHEMAS[type].parse({ type })),
  },
  inputProps: { resourceName: "framing", children: Task.selectItems(TYPE_NAMES) },
});

const ByteOrderField = Form.buildSelectField<library.ByteOrder>({
  inputProps: {
    resourceName: "byte order",
    children: Task.selectItems(BYTE_ORDER_NAMES),
  },
});

const ChecksumField = Form.buildSelectField<bus.Checksum>({
  fieldKey: "checksum",
  fieldProps: { label: "Checksum" },
  inputProps: { resourceName: "checksum", children: Task.selectItems(CHECKSUM_NAMES) },
});

const LengthSizeField = Form.buildSelectField<LengthSize>({
  fieldKey: "lengthSize",
  fieldProps: { label: "Length size" },
  inputProps: {
    resourceName: "length size",
    // Numeric keys, so the items cannot come from a name record.
    children: LENGTH_SIZES.map((size) => (
      <Select.Item<LengthSize> key={size} itemKey={size}>
        {size === 1 ? "1 byte" : `${size} bytes`}
      </Select.Item>
    )),
  },
});

const DELIMITER_INPUT_PROPS = { placeholder: "\\n", onlyChangeOnBlur: true } as const;

/** Edits the delimiter as escaped text, so a newline reads as \n. */
const DelimiterField = (): ReactElement => (
  <Form.Field<string> path={`${PATH}.delimiter`} label="Delimiter">
    {({ value, onChange, ...rest }) => (
      <Input.Text
        {...rest}
        {...DELIMITER_INPUT_PROPS}
        value={formatEscaped(value)}
        onChange={(v) => onChange(parseEscaped(v))}
      />
    )}
  </Form.Field>
);

const SYNC_INPUT_PROPS = { placeholder: "AA55" } as const;

const SyncFields = (): ReactElement => {
  const checksum = Form.useFieldValue<bus.Checksum>(`${PATH}.checksum`);
  return (
    <>
      <Form.TextField
        path={`${PATH}.sync`}
        label="Sync"
        inputProps={SYNC_INPUT_PROPS}
      />
      <Form.NumericField path={`${PATH}.lengthOffset`} label="Length offset" />
      <LengthSizeField path={PATH} />
      <ByteOrderField path={`${PATH}.byteOrder`} label="Length byte order" />
      <Form.NumericField path={`${PATH}.lengthAdjustment`} label="Length adjustment" />
      <ChecksumField path={PATH} />
      {checksum !== "none" && (
        <ByteOrderField
          path={`${PATH}.checksumByteOrder`}
          label="Checksum byte order"
        />
      )}
    </>
  );
};

/** Edits how a byte stream task splits its stream into messages. */
export const Framing = (): ReactElement => {
  const type = Form.useFieldValue<bus.FramingType>(`${PATH}.type`);
  return (
    <Flex.Box x wrap>
      <TypeField path={`${PATH}.type`} />
      {type === "delimiter" && <DelimiterField />}
      {type === "fixed" && <Form.NumericField path={`${PATH}.length`} label="Length" />}
      {type === "sync" && <SyncFields />}
    </Flex.Box>
  );
};
