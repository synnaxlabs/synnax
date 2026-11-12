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
