// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/secret/Field.css";

import { NotFoundError, secret } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, Secret } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/platform/button";
import { CSS } from "@/platform/css";
import { useCreateModal } from "@/platform/secret/useCreateModal";
import { Validity } from "@/platform/secret/validity";

const ListItem = (props: List.ItemRenderProps<secret.Key>): ReactElement | null => {
  const s = List.useItem<secret.Key, secret.Secret>(props.itemKey);
  if (s == null) return null;
  return (
    <Select.Item {...props} align="center" gap="medium">
      <Icon.Secret className={CSS.BE("secret-field", "icon")} />
      <Flex.Box y empty grow className={CSS.BE("secret-field", "text")}>
        <Text.Text overflow="fade">{s.name}</Text.Text>
        <Text.Text level="tiny" color={9} overflow="fade">
          {Validity.describeSecret(s)}
        </Text.Text>
      </Flex.Box>
    </Select.Item>
  );
};

const listItem = Component.renderProp(ListItem);

interface TriggerProps {
  value: secret.Key;
  preview?: boolean;
}

// The text of a trigger with no secret to name. It is empty while the secret loads.
const placeholderOf = (
  result: Flux.Result<secret.Secret>,
  preview?: boolean,
): string => {
  if (result.variant === "disabled")
    return preview === true ? "None" : "Select a secret";
  if (result.variant !== "error") return "";
  const { error } = result.status.details;
  return Flux.DeletedError.matches(error) || NotFoundError.matches(error.cause)
    ? "Secret not found"
    : "Unavailable secret";
};

const Trigger = ({ value, preview }: TriggerProps): ReactElement => {
  const result = Secret.useResult(primitive.isZero(value) ? null : { key: value });
  const itemStatus = Input.useItemStatus({});
  return (
    <Dialog.Trigger
      variant="outlined"
      gap="small"
      grow
      preview={preview}
      className={CSS.BE("secret-field", "trigger")}
      {...itemStatus}
    >
      <Icon.Secret className={CSS.BE("secret-field", "icon")} />
      {result.variant !== "success" ? (
        <Text.Text el="span" color={8} className={CSS.BE("secret-field", "text")}>
          {placeholderOf(result, preview)}
        </Text.Text>
      ) : (
        <>
          <Text.Text
            el="span"
            overflow="fade"
            className={CSS.BE("secret-field", "text")}
          >
            {result.data.name}
          </Text.Text>
          <Text.Text
            el="span"
            level="tiny"
            color={9}
            className={CSS.BE("secret-field", "date")}
          >
            {Validity.describeSecret(result.data)}
          </Text.Text>
        </>
      )}
    </Dialog.Trigger>
  );
};

interface CreateButtonProps {
  defaultName: string;
  onCreate: (key: secret.Key) => void;
}

const CreateButton = ({ defaultName, onCreate }: CreateButtonProps): ReactElement => {
  const openCreate = useCreateModal();
  const { close } = Dialog.useContext();
  const handleClick = async () => {
    close();
    const key = await openCreate({ name: defaultName });
    if (key != null) onCreate(key);
  };
  return (
    <Button.CreateListItem size="small" onClick={() => void handleClick()}>
      New secret
    </Button.CreateListItem>
  );
};

interface PickerProps extends Input.Control<secret.Key> {
  defaultName: string;
  preview?: boolean;
}

const Picker = ({
  value,
  onChange,
  defaultName,
  preview,
}: PickerProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Secret.useList();
  const canCreate = Access.useCreateGranted(secret.TYPE_RESOURCE_ID);
  return (
    <Dialog.Frame variant="connected" grow>
      <Select.Frame<secret.Key, secret.Secret | undefined>
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={more}
        value={value}
        onChange={onChange}
        closeDialogOnSelect
      >
        <Trigger value={value} preview={preview} />
        <Select.Dialog className={CSS.BE("secret-field", "dialog")}>
          <Select.Body<secret.Key>
            resourceName="secret"
            onSearch={search}
            status={status}
            footer={
              canCreate && (
                <CreateButton defaultName={defaultName} onCreate={onChange} />
              )
            }
          >
            {listItem}
          </Select.Body>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
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
      <Picker
        value={value}
        onChange={onChange}
        defaultName={defaultName}
        preview={preview}
      />
    )}
  </Form.Field>
);
