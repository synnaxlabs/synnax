// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { type deep, type payload } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form as PlatformForm } from "@/platform/form";

export interface EnumMappingSectionProps {
  /** The path of the enum values. */
  path: deep.Target<PlatformForm.Entry<"label", number>[]>;
  /** Puts the channel value before its label, for a write that sends the label. */
  valueFirst?: boolean;
}

/** Maps string labels in a payload to numeric channel values. */
export const EnumMappingSection = ({
  path,
  valueFirst,
}: EnumMappingSectionProps): ReactElement => (
  <Form.Section title="Enum mapping">
    <PlatformForm.KeyValueEditor
      path={path}
      resourceName="mapping"
      keyField="label"
      keyPlaceholder="String (e.g. ON)"
      valueType="number"
      valueFirst={valueFirst}
    />
  </Form.Section>
);

export interface TextEnumMappingSectionProps {
  /** The path of the enum values. */
  path: deep.Target<PlatformForm.Entry<"label", string>[]>;
  /** Shows what a channel value looks like, such as "1". */
  valuePlaceholder?: string;
}

/**
 * Maps string labels in a payload to channel values held as text: strings, or the
 * decimal digits of 64-bit integers.
 */
export const TextEnumMappingSection = ({
  path,
  valuePlaceholder,
}: TextEnumMappingSectionProps): ReactElement => (
  <Form.Section title="Enum mapping">
    <PlatformForm.KeyValueEditor
      path={path}
      resourceName="mapping"
      keyField="label"
      keyPlaceholder="String (e.g. ON)"
      valuePlaceholder={valuePlaceholder}
      valueType="string"
    />
  </Form.Section>
);

export interface BooleanEnumMappingSectionProps {
  /** The path of the enum values. */
  path: deep.Target<payload.BooleanEntry[]>;
}

/**
 * Maps string labels in a payload to boolean channel values, shown as 1 for true and 0
 * for false. Any value but 0 is true.
 */
export const BooleanEnumMappingSection = ({
  path,
}: BooleanEnumMappingSectionProps): ReactElement => {
  const { set } = Form.useContext();
  const entries = Form.useFieldValue(path);
  const rows = entries.map(({ label, value }) => [label, value ? 1 : 0]);
  const handleChange = (next: Input.TableCell[][]) =>
    set(
      path,
      next.map(([label, value]) => ({ label: String(label), value: value !== 0 })),
    );
  return (
    <Form.Section title="Enum mapping">
      <Input.Item padHelpText={false}>
        <Input.Table value={rows} onChange={handleChange} resourceName="mapping">
          <Input.TableColumn name="Label" type="string">
            {(p) => <Input.Text {...p} placeholder="String (e.g. ON)" />}
          </Input.TableColumn>
          <Input.TableColumn name="Value" />
        </Input.Table>
      </Input.Item>
    </Form.Section>
  );
};
