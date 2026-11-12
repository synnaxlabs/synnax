// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { type deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form as PlatformForm } from "@/platform/form";

interface NumberEntries {
  /** The path of the enum values. */
  path: deep.Target<PlatformForm.Entry<"label", number>[]>;
  valueType?: "number";
}

interface TextEntries {
  /** The path of the enum values, held as text: strings, or 64-bit integers. */
  path: deep.Target<PlatformForm.Entry<"label", string>[]>;
  valueType: "string";
  /** The value of an added entry, such as "0" for an integer. Defaults to "". */
  defaultValue?: string;
  /** Shows what a channel value looks like, such as "1". */
  valuePlaceholder?: string;
}

export type EnumMappingSectionProps = (NumberEntries | TextEntries) & {
  /** Puts the channel value before its label, for a write that sends the label. */
  valueFirst?: boolean;
};

/**
 * @returns a label that none of labels holds, for an added enum entry. The Core rejects
 * a repeated or empty label.
 */
export const createLabel = (labels: string[]): string => {
  let n = labels.length + 1;
  while (labels.includes(`LABEL_${n}`)) n++;
  return `LABEL_${n}`;
};

const createEntry =
  <V,>(value: V) =>
  (labels: string[]): { key: string; value: V } => ({
    key: createLabel(labels),
    value,
  });

/** Maps string labels in a payload to channel values. */
export const EnumMappingSection = ({
  valueFirst,
  ...rest
}: EnumMappingSectionProps): ReactElement => (
  <Form.Section title="Enum mapping">
    {rest.valueType === "string" ? (
      <PlatformForm.KeyValueEditor
        path={rest.path}
        resourceName="mapping"
        keyField="label"
        keyPlaceholder="String (e.g. ON)"
        valuePlaceholder={rest.valuePlaceholder}
        valueType="string"
        valueFirst={valueFirst}
        createEntry={createEntry(rest.defaultValue ?? "")}
      />
    ) : (
      <PlatformForm.KeyValueEditor
        path={rest.path}
        resourceName="mapping"
        keyField="label"
        keyPlaceholder="String (e.g. ON)"
        valueType="number"
        valueFirst={valueFirst}
        createEntry={createEntry(0)}
      />
    )}
  </Form.Section>
);
