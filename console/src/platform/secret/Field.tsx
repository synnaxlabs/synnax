// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, secret } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Input } from "@synnaxlabs/lyra/input";
import { Access, Secret } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Picker } from "@/platform/picker";
import { useCreateModal } from "@/platform/secret/useCreateModal";
import { Validity } from "@/platform/secret/validity";

const describe = (s: secret.Secret): Picker.Description => ({
  text: Validity.describeSecret(s),
});

/** Describes a held file item of another rack, which the device cannot read. */
export const OTHER_RACK_DESCRIPTION: Picker.Description = {
  text: "File on another rack",
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
}

const Control = ({ value, defaultName, rack, ...rest }: ControlProps): ReactElement => {
  const list = Secret.useList({ query: { where: whereOnRack(rack) } });
  const result = Secret.useResult(primitive.isZero(value) ? null : { key: value });
  const canCreate = Access.useCreateGranted(secret.TYPE_RESOURCE_ID);
  const openCreate = useCreateModal();
  const describeOnRack = (s: secret.Secret): Picker.Description =>
    s.source === "file" && s.rack !== rack ? OTHER_RACK_DESCRIPTION : describe(s);
  return (
    <Picker.Select<secret.Key, secret.Secret>
      value={value}
      resourceName="secret"
      icon={Icon.Secret}
      describe={describeOnRack}
      list={list}
      result={result}
      onCreate={
        canCreate
          ? (name) => openCreate({ name: name === "" ? defaultName : name, rack })
          : undefined
      }
      {...rest}
    />
  );
};

export interface FieldProps extends Omit<Form.FieldProps<secret.Key>, "children"> {
  /** The name a created secret starts with, such as "Line 2 alerts routing key". */
  defaultName: string;
  /** The rack of the device. The field lists only the file items of this rack. */
  rack: rack.Key;
}

/**
 * A form field that holds the key of a secret. It selects a secret, or creates one that
 * starts with the default name and selects it.
 */
export const Field = ({ defaultName, rack, ...rest }: FieldProps): ReactElement => (
  <Form.Field<secret.Key> {...rest}>
    {({ value, onChange, preview }) => (
      <Control
        value={value}
        onChange={onChange}
        defaultName={defaultName}
        rack={rack}
        preview={preview}
      />
    )}
  </Form.Field>
);
