// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, secret } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import type { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Secret } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Picker } from "@/platform/picker";
import { useCreateModal } from "@/platform/secret/useCreateModal";
import { Validity } from "@/platform/secret/validity";

const describe = (s: secret.Secret): Picker.Description => ({
  text: Validity.describeSecret(s),
});

/** Describes a held file item of another rack, which the device cannot read. */
export const OTHER_RACK_DESCRIPTION: Picker.Description = {
  text: "File on another Driver",
  status: "error",
};

const STORED: secret.Where = { source: { eq: "stored" } };

/**
 * Matches stored items and the file items of a rack, the only file items its devices
 * can read. A zero rack matches only stored items.
 */
export const whereOnRack = (rack: rack.Key): secret.Where =>
  primitive.isZero(rack) ? STORED : { or: [STORED, { racks: [rack] }] };

interface ControlProps extends Input.Control<secret.Key> {
  defaultName: string;
  rack: rack.Key;
  preview?: boolean;
  allowNone: boolean;
}

const Control = ({
  value,
  onChange,
  defaultName,
  rack,
  allowNone,
  preview,
}: ControlProps): ReactElement => {
  const list = Secret.useList({ query: { where: whereOnRack(rack) } });
  const result = Secret.useResult(primitive.isZero(value) ? null : { key: value });
  const canCreate = Access.useCreateGranted(secret.TYPE_RESOURCE_ID);
  const openCreate = useCreateModal();
  const onOtherRack =
    result.variant === "success" &&
    result.data.source === "file" &&
    result.data.rack !== rack;
  // A cleared field holds the zero key, as one never set does.
  const selection = allowNone
    ? {
        value,
        onChange: (key: secret.Key | null) => onChange(key ?? ""),
        allowNone: true as const,
      }
    : { value, onChange, allowNone: false as const };
  return (
    <Flex.Box y gap="small" grow>
      <Picker.Select<secret.Secret>
        resourceName="secret"
        icon={Icon.Secret}
        describe={describe}
        list={list}
        result={result}
        onCreate={
          canCreate
            ? (name) => openCreate({ name: name === "" ? defaultName : name, rack })
            : undefined
        }
        preview={preview}
        {...selection}
      />
      {onOtherRack && (
        <Picker.Line>
          <Text.Text level="small" status={OTHER_RACK_DESCRIPTION.status}>
            {OTHER_RACK_DESCRIPTION.text}
          </Text.Text>
        </Picker.Line>
      )}
    </Flex.Box>
  );
};

export interface FieldProps extends Omit<Form.FieldProps<secret.Key>, "children"> {
  /** The name a created secret starts with, such as "Line 2 alerts routing key". */
  defaultName: string;
  /** The rack of the device. The field lists only the file items of this rack. */
  rack: rack.Key;
  /** Whether the user may clear the field, which then holds no secret. */
  allowNone?: boolean;
}

/**
 * A form field that holds the key of a secret. It selects a secret, or creates one that
 * starts with the default name and selects it.
 */
export const Field = ({
  defaultName,
  rack,
  allowNone = false,
  ...rest
}: FieldProps): ReactElement => (
  <Form.Field<secret.Key> {...rest}>
    {({ value, onChange, preview }) => (
      <Control
        value={value}
        onChange={onChange}
        defaultName={defaultName}
        rack={rack}
        preview={preview}
        allowNone={allowNone}
      />
    )}
  </Form.Field>
);
