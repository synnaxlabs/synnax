// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/certificate/Field.css";

import { certificate, NotFoundError } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Certificate, Flux } from "@synnaxlabs/pluto";
import { primitive, TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/platform/button";
import { Details } from "@/platform/certificate/Details";
import { useCreateModal } from "@/platform/certificate/useCreateModal";
import { CSS } from "@/platform/css";
import { Secret } from "@/platform/secret";

const nextMinute = (now: TimeStamp): TimeStamp => now.add(TimeSpan.MINUTE);

const ListItem = (
  props: List.ItemRenderProps<certificate.Key>,
): ReactElement | null => {
  const c = List.useItem<certificate.Key, certificate.Certificate>(props.itemKey);
  const now = useNow(nextMinute);
  if (c == null) return null;
  const { text, status } = Secret.Validity.describeCertificate(c, now);
  return (
    <Select.Item {...props} align="center" gap="medium">
      <Icon.Certificate className={CSS.BE("certificate-field", "icon")} />
      <Flex.Box y empty grow className={CSS.BE("certificate-field", "text")}>
        <Text.Text overflow="fade">{c.name}</Text.Text>
        <Text.Text level="tiny" color={9} status={status} overflow="fade">
          {text}
        </Text.Text>
      </Flex.Box>
    </Select.Item>
  );
};

const listItem = Component.renderProp(ListItem);

// The text of a trigger with no certificate to name. It is empty while one loads.
const placeholderOf = (
  result: Flux.Result<certificate.Certificate>,
  preview?: boolean,
): string => {
  if (result.variant === "disabled")
    return preview === true ? "None" : "Select a certificate";
  if (result.variant !== "error") return "";
  const { error } = result.status.details;
  return Flux.DeletedError.matches(error) || NotFoundError.matches(error.cause)
    ? "Certificate not found"
    : "Unavailable certificate";
};

interface TriggerProps {
  result: Flux.Result<certificate.Certificate>;
  preview?: boolean;
}

const Trigger = ({ result, preview }: TriggerProps): ReactElement => {
  const itemStatus = Input.useItemStatus({});
  const now = useNow(nextMinute);
  let content: ReactElement;
  if (result.variant !== "success")
    content = (
      <Text.Text el="span" color={8} className={CSS.BE("certificate-field", "text")}>
        {placeholderOf(result, preview)}
      </Text.Text>
    );
  else {
    const { text, status } = Secret.Validity.describeCertificate(result.data, now);
    content = (
      <>
        <Text.Text
          el="span"
          overflow="fade"
          className={CSS.BE("certificate-field", "text")}
        >
          {result.data.name}
        </Text.Text>
        <Text.Text
          el="span"
          level="tiny"
          color={9}
          status={status}
          className={CSS.BE("certificate-field", "validity")}
        >
          {text}
        </Text.Text>
      </>
    );
  }
  return (
    <Dialog.Trigger
      variant="outlined"
      gap="small"
      grow
      preview={preview}
      className={CSS.BE("certificate-field", "trigger")}
      {...itemStatus}
    >
      <Icon.Certificate className={CSS.BE("certificate-field", "icon")} />
      {content}
    </Dialog.Trigger>
  );
};

interface CreateButtonProps {
  defaultName: string;
  onCreate: (key: certificate.Key) => void;
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
      New certificate
    </Button.CreateListItem>
  );
};

interface PickerProps extends Input.Control<certificate.Key> {
  defaultName: string;
  preview?: boolean;
}

const Picker = ({
  value,
  onChange,
  defaultName,
  preview,
}: PickerProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Certificate.useList();
  const result = Certificate.useResult(primitive.isZero(value) ? null : { key: value });
  const canCreate = Access.useCreateGranted(certificate.TYPE_RESOURCE_ID);
  return (
    <Flex.Box y gap="small" grow>
      <Dialog.Frame variant="connected">
        <Select.Frame<certificate.Key, certificate.Certificate | undefined>
          data={data}
          getItem={getItem}
          subscribe={subscribe}
          onFetchMore={more}
          value={value}
          onChange={onChange}
          closeDialogOnSelect
        >
          <Trigger result={result} preview={preview} />
          <Select.Dialog className={CSS.BE("certificate-field", "dialog")}>
            <Select.Body<certificate.Key>
              resourceName="certificate"
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
      {result.variant === "success" && <Details certificate={result.data} />}
    </Flex.Box>
  );
};

export interface FieldProps extends Omit<Form.FieldProps<certificate.Key>, "children"> {
  /** The name a created certificate starts with, such as "Plant CA certificate". */
  defaultName: string;
}

/**
 * A form field that holds the key of a certificate and shows its details. It selects a
 * certificate, or creates one that starts with the default name and selects it.
 */
export const Field = ({ defaultName, ...rest }: FieldProps): ReactElement => (
  <Form.Field<certificate.Key> {...rest}>
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
