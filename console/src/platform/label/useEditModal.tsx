// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/label/Edit.css";

import { label } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Component } from "@synnaxlabs/lyra/component";
import { CSS as PCSS } from "@synnaxlabs/lyra/css";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, type Flux, Label } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { useEffect, useRef, useState } from "react";

import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

interface FieldsProps {
  inputProps?: Pick<Input.TextProps, "ref" | "onlyChangeOnBlur">;
}

const Fields = ({ inputProps }: FieldsProps) => (
  <Flex.Box x gap="small" align="center">
    <Form.Field<color.Color> path="color" padHelpText={false} showLabel={false}>
      {(p) => <Color.Swatch onlyChangeOnBlur variant="text" {...p} />}
    </Form.Field>
    <Form.TextField
      showLabel={false}
      path="name"
      showHelpText={false}
      padHelpText={false}
      inputProps={{
        placeholder: "Name",
        variant: "text",
        selectOnFocus: true,
        resetOnBlurIfEmpty: true,
        onlyChangeOnBlur: true,
        ...inputProps,
      }}
    />
  </Flex.Box>
);

const Item = (props: List.ItemProps<label.Key>) => {
  const { itemKey } = props;
  const { form } = Label.useForm({ query: { key: itemKey } });
  const { update: handleDelete } = Label.useDelete();
  const canDelete = Access.useDeleteGranted(label.ontologyID(itemKey));
  return (
    <List.Item
      className={CSS.BE("label", "list-item")}
      align="center"
      justify="between"
      {...props}
    >
      <Form.Form<typeof Label.formSchema> {...form}>
        <Fields />
      </Form.Form>
      {canDelete && (
        <Button.Button
          variant="outlined"
          size="small"
          reveal
          onClick={() => handleDelete(itemKey)}
        >
          <Icon.Delete />
        </Button.Button>
      )}
    </List.Item>
  );
};

interface CreateItemProps {
  visible: boolean;
  onClose: () => void;
}

const CreateItem = ({ visible, onClose }: CreateItemProps) => {
  const { form, save } = Label.useCreateForm({
    query: null,
    afterSave: ({
      reset,
    }: Flux.AfterSaveParams<null, typeof Label.createFormSchema>) => {
      onClose();
      reset({ name: "", color: color.construct("#000000") });
    },
  });
  const inputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (visible) inputRef.current?.focus();
  }, [visible]);
  const ref = useRef<HTMLDivElement>(null);
  Dialog.useClickOutside({
    ref,
    onClickOutside: () => {
      if (form.validate()) save();
      else onClose();
    },
  });
  return (
    <List.Item
      ref={ref}
      index={0}
      itemKey=""
      className={CSS.cls(
        CSS.BE("label", "list-item"),
        CSS.M("create"),
        PCSS.visible(visible),
      )}
      align="center"
      justify="between"
    >
      <Form.Form<typeof Label.createFormSchema> {...form}>
        <Fields inputProps={{ ref: inputRef, onlyChangeOnBlur: false }} />
      </Form.Form>
      <Flex.Box pack>
        <Button.Button
          variant="filled"
          size="small"
          onClick={() => save()}
          trigger={visible ? ["Enter"] : undefined}
          triggerIndicator
        >
          <Icon.Check />
        </Button.Button>
        <Button.Button variant="outlined" size="small" onClick={onClose}>
          <Icon.Close />
        </Button.Button>
      </Flex.Box>
    </List.Item>
  );
};

const listItem = Component.renderProp(Item);

const EMPTY_CONTENT = (
  <Flex.Box center>
    <Text.Text level="h4" color={9}>
      No labels created
    </Text.Text>
  </Flex.Box>
);

const Edit = () => {
  const { data, getItem, retrieve, subscribe, answered } = Label.useList();
  const { fetchMore, search } = List.usePager({ retrieve, pageSize: 15 });
  const [newFormVisible, setNewFormVisible] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const hasCreatePermission = Access.useCreateGranted(label.TYPE_ONTOLOGY_ID);
  return (
    <Modal.Frame y className={CSS.BE("label", "edit")}>
      <Modal.Header icon={<Icon.Label />}>Label.Edit</Modal.Header>
      <List.Frame<label.Key, label.Label>
        data={data}
        getItem={getItem}
        onFetchMore={fetchMore}
        subscribe={subscribe}
      >
        <Input.Text
          value={searchTerm}
          onChange={(v) => {
            setSearchTerm(v);
            search(v);
          }}
          placeholder="Search labels..."
          startContent={<Icon.Search />}
          autoFocus
          flush
          size="large"
          full="x"
          className={CSS.BE("label", "search")}
        />
        <Divider.Divider x />
        <Flex.Box y className={CSS.BE("label", "items-container")} empty>
          <CreateItem
            visible={newFormVisible}
            onClose={() => setNewFormVisible(false)}
          />
          <List.Scroll grow>
            <List.Items emptyContent={answered && !newFormVisible && EMPTY_CONTENT}>
              {listItem}
            </List.Items>
          </List.Scroll>
          {!newFormVisible && hasCreatePermission && (
            <PlatformButton.CreateListItem
              onClick={() => setNewFormVisible(true)}
              className={CSS.BE("label", "create")}
            >
              New label
            </PlatformButton.CreateListItem>
          )}
        </Flex.Box>
      </List.Frame>
    </Modal.Frame>
  );
};

export const useEditModal = Modals.create(Edit);
