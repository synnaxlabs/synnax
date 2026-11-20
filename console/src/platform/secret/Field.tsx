// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { secret } from "@synnaxlabs/client";
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

interface ControlProps extends Input.Control<secret.Key> {
  defaultName: string;
  preview?: boolean;
}

const Control = ({ value, defaultName, ...rest }: ControlProps): ReactElement => {
  const list = Secret.useList();
  const result = Secret.useResult(primitive.isZero(value) ? null : { key: value });
  const canCreate = Access.useCreateGranted(secret.TYPE_RESOURCE_ID);
  const openCreate = useCreateModal();
  return (
    <Picker.Select<secret.Key, secret.Secret>
      value={value}
      resourceName="secret"
      icon={Icon.Secret}
      describe={describe}
      list={list}
      result={result}
      onCreate={
        canCreate
          ? (name) => openCreate({ name: name === "" ? defaultName : name })
          : undefined
      }
      {...rest}
    />
  );
};

export interface FieldProps extends Omit<Form.FieldProps<secret.Key>, "children"> {
  /** The name a created secret starts with, such as "Line 2 alerts routing key". */
  defaultName: string;
}

/**
 * A form field that holds the key of a secret. It selects a secret, or creates one that
 * starts with the default name and selects it.
 */
export const Field = ({ defaultName, ...rest }: FieldProps): ReactElement => (
  <Form.Field<secret.Key> {...rest}>
    {({ value, onChange, preview }) => (
      <Control
        value={value}
        onChange={onChange}
        defaultName={defaultName}
        preview={preview}
      />
    )}
  </Form.Field>
);
