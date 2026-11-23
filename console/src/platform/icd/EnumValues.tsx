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
import { deep, type payload, type xy } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Form as PlatformForm } from "@/platform/form";

/** A field whose kind decides the values its labels map to. */
type Enumerable =
  | { type: "float"; enumValues: payload.FloatEntry[] }
  | { type: "integer"; enumValues: payload.IntegerEntry[] }
  | { type: "unsigned"; enumValues: payload.UnsignedEntry[] }
  | { type: "string"; enumValues: payload.StringEntry[] }
  | { type: "boolean"; enumValues: payload.BooleanEntry[] }
  | { type: "timestamp" | "json" | "uuid" | "bytes" };

const LABEL_PLACEHOLDER = "String (e.g. ON)";

interface TextEntriesProps extends Pick<Input.ItemProps, "label"> {
  path: deep.Target<PlatformForm.Entry<"label", string>[]>;
  /** Shows what a value looks like, such as "1". */
  valuePlaceholder: string;
}

/** Maps labels to values held as text: strings, or the digits of 64-bit integers. */
const TextEntries = ({
  path,
  label,
  valuePlaceholder,
}: TextEntriesProps): ReactElement => (
  <PlatformForm.KeyValueEditor
    path={path}
    label={label}
    resourceName="mapping"
    keyField="label"
    keyPlaceholder={LABEL_PLACEHOLDER}
    valuePlaceholder={valuePlaceholder}
    valueType="string"
  />
);

const CELL_FIELDS = ["label", "value"];

interface BooleanEntriesProps extends Pick<Input.ItemProps, "label"> {
  path: deep.Target<payload.BooleanEntry[]>;
}

/** Maps labels to booleans, shown as 1 for true and 0 for false. Any but 0 is true. */
const BooleanEntries = ({ path, label }: BooleanEntriesProps): ReactElement => {
  const { set, mode } = Form.useContext();
  const entries = Form.useFieldValue(path);
  const statuses = Form.useStatuses(path);
  const rows = entries.map(({ label, value }) => [label, value ? 1 : 0]);
  const cellStatus = ({ x, y }: xy.XY) =>
    statuses.find(({ key }) => key === `${String(path)}.${y}.${CELL_FIELDS[x]}`);
  const handleChange = (next: Input.TableCell[][]) =>
    set(
      path,
      next.map(([label, value]) => ({ label: String(label), value: value !== 0 })),
    );
  return (
    <Input.Item label={label} padHelpText={false}>
      <Input.Table
        value={rows}
        onChange={handleChange}
        resourceName="mapping"
        cellStatus={cellStatus}
        preview={mode === "preview" ? true : undefined}
      >
        <Input.TableColumn name="Label" type="string">
          {(p) => <Input.Text {...p} placeholder={LABEL_PLACEHOLDER} />}
        </Input.TableColumn>
        <Input.TableColumn name="Value" />
      </Input.Table>
    </Input.Item>
  );
};

export interface EnumValuesProps extends Pick<Input.ItemProps, "label"> {
  /** The path of the field. */
  path: deep.Target<Enumerable>;
}

/**
 * Maps text labels in a payload to the values of the field's kind. Renders nothing for
 * a kind that maps no labels.
 */
export const EnumValues = ({ path, label }: EnumValuesProps): ReactElement | null => {
  const type = Form.useFieldValue(deep.widen(path).type);
  switch (type) {
    case "float":
      return (
        <PlatformForm.KeyValueEditor
          path={deep.variant(path, "type", type).enumValues}
          label={label}
          resourceName="mapping"
          keyField="label"
          keyPlaceholder={LABEL_PLACEHOLDER}
          valueType="number"
        />
      );
    case "integer":
    case "unsigned":
      return (
        <TextEntries
          path={deep.variant(path, "type", type).enumValues}
          label={label}
          valuePlaceholder="Integer (e.g. 1)"
        />
      );
    case "string":
      return (
        <TextEntries
          path={deep.variant(path, "type", type).enumValues}
          label={label}
          valuePlaceholder="String (e.g. running)"
        />
      );
    case "boolean":
      return (
        <BooleanEntries
          path={deep.variant(path, "type", type).enumValues}
          label={label}
        />
      );
    default:
      return null;
  }
};
