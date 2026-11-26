// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { ICD } from "@synnaxlabs/pluto";
import { deep, record } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import {
  formatEscaped,
  formatHexBytes,
  FormattedField,
  parseEscaped,
  parseHexBytes,
} from "@/feature/icd/editor/FormattedField";
import { CSS } from "@/platform/css";

type Type = icd.FramingType | "none";

const NAMES: Record<Type, string> = {
  none: "None",
  delimited: "Delimited",
  fixed: "Fixed length",
  sync: "Sync and length",
  cobs: "COBS",
  slip: "SLIP",
};

const TYPES = record.keys(NAMES);

const CHECKSUM_NAMES: Record<icd.ChecksumAlgorithm | "none", string> = {
  none: "None",
  crc16_ccitt_false: "CRC-16/CCITT-FALSE",
  crc16_modbus: "CRC-16/MODBUS",
  crc32: "CRC-32",
};

const CHECKSUMS = record.keys(CHECKSUM_NAMES);

const BYTES = { endContent: "bytes" };

const create = (type: icd.FramingType): icd.Framing => {
  switch (type) {
    case "delimited":
      return { type, delimiter: "\n" };
    case "fixed":
      return { type, length: 1 };
    case "sync":
      return {
        type,
        sync: new Uint8Array([0xaa, 0x55]),
        lengthOffset: 2,
        lengthSize: "one",
        byteOrder: "little_endian",
        lengthAdjustment: 0,
        maxLength: 65536,
      };
    case "cobs":
    case "slip":
      return { type };
  }
};

interface ByteOrderProps {
  path: deep.Target<icd.ByteOrder>;
  label: string;
}

const ByteOrder = ({ path, label }: ByteOrderProps): ReactElement => (
  <Form.Field<icd.ByteOrder> path={path} label={label}>
    {(p) => (
      <Select.Buttons<icd.ByteOrder> {...p}>
        <Select.Item<icd.ByteOrder> itemKey="little_endian">Little endian</Select.Item>
        <Select.Item<icd.ByteOrder> itemKey="big_endian">Big endian</Select.Item>
      </Select.Buttons>
    )}
  </Form.Field>
);

interface ChecksumProps {
  path: deep.Target<icd.Checksum | undefined>;
}

const Checksum = ({ path }: ChecksumProps): ReactElement => {
  const { set, mode } = Form.useContext();
  const checksum = Form.useFieldValue(path, { optional: true });
  const handleChange = (next: icd.ChecksumAlgorithm | "none") =>
    set(
      path,
      next === "none"
        ? undefined
        : { algorithm: next, byteOrder: checksum?.byteOrder ?? "little_endian" },
    );
  return (
    <Flex.Box x>
      <Input.Item label="Checksum" padHelpText>
        <Select.Simple<icd.ChecksumAlgorithm | "none">
          resourceName="checksum"
          value={checksum?.algorithm ?? "none"}
          onChange={handleChange}
          variant="floating"
          allowNone={false}
          preview={mode === "preview"}
        >
          {CHECKSUMS.map((c) => (
            <Select.Item key={c} itemKey={c}>
              {CHECKSUM_NAMES[c]}
            </Select.Item>
          ))}
        </Select.Simple>
      </Input.Item>
      {checksum != null && (
        <ByteOrder path={deep.present(path).byteOrder} label="Checksum byte order" />
      )}
    </Flex.Box>
  );
};

interface SyncFieldsProps {
  path: deep.Path<icd.SyncFraming>;
}

const SyncFields = ({ path }: SyncFieldsProps): ReactElement => (
  <>
    <Flex.Box x>
      <FormattedField<Uint8Array>
        path={path.sync}
        label="Sync bytes"
        placeholder="AA 55"
        format={formatHexBytes}
        parse={parseHexBytes}
      />
      <Form.NumericField path={path.maxLength} label="Max length" inputProps={BYTES} />
    </Flex.Box>
    <Flex.Box x>
      <Form.NumericField
        path={path.lengthOffset}
        label="Length offset"
        inputProps={BYTES}
      />
      <Form.Field<icd.LengthSize> path={path.lengthSize} label="Length size">
        {(p) => (
          <Select.Buttons<icd.LengthSize> {...p}>
            <Select.Item<icd.LengthSize> itemKey="one">1 byte</Select.Item>
            <Select.Item<icd.LengthSize> itemKey="two">2 bytes</Select.Item>
            <Select.Item<icd.LengthSize> itemKey="four">4 bytes</Select.Item>
          </Select.Buttons>
        )}
      </Form.Field>
    </Flex.Box>
    <Flex.Box x>
      <ByteOrder path={path.byteOrder} label="Length byte order" />
      <Form.NumericField
        path={path.lengthAdjustment}
        label="Length adjustment"
        inputProps={BYTES}
      />
    </Flex.Box>
    <Checksum path={path.checksum} />
  </>
);

/**
 * Edits how a byte stream splits into the messages of the ICD. A type switch replaces
 * the framing with a new one of that type.
 */
export const Framing = (): ReactElement => {
  const path = ICD.formPath.framing;
  const { set, mode } = Form.useContext();
  const type = Form.useFieldValue(deep.present(path).type, { optional: true });
  const handleChange = (next: Type) => {
    if (next !== (type ?? "none"))
      set(path, next === "none" ? undefined : create(next));
  };
  return (
    <Flex.Box y bordered empty className={CSS.BE("icd-editor", "framing")}>
      <Form.Sections>
        <Form.Section title="Framing">
          <Flex.Box x>
            <Input.Item label="Type" padHelpText>
              <Select.Simple<Type>
                resourceName="framing"
                value={type ?? "none"}
                onChange={handleChange}
                variant="floating"
                allowNone={false}
                preview={mode === "preview"}
              >
                {TYPES.map((t) => (
                  <Select.Item key={t} itemKey={t}>
                    {NAMES[t]}
                  </Select.Item>
                ))}
              </Select.Simple>
            </Input.Item>
            {type === "delimited" && (
              <FormattedField<string>
                path={deep.variant(path, "type", "delimited").delimiter}
                label="Delimiter"
                placeholder="\n"
                format={formatEscaped}
                parse={parseEscaped}
              />
            )}
            {type === "fixed" && (
              <Form.NumericField
                path={deep.variant(path, "type", "fixed").length}
                label="Length"
                inputProps={BYTES}
              />
            )}
          </Flex.Box>
          {type === "sync" && <SyncFields path={deep.variant(path, "type", "sync")} />}
        </Form.Section>
      </Form.Sections>
    </Flex.Box>
  );
};
