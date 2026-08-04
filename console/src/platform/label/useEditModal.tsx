// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/label/Edit.css";

import { type label, type query } from "@synnaxlabs/client";
import {
  Button,
  Color,
  Component,
  CSS as PCSS,
  Divider,
  Flex,
  type Flux,
  Form,
  Icon,
  Input,
  Label,
  List,
  Text,
  useClickOutside,
} from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef, useState } from "react";

import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

interface LabelListItemProps extends List.ItemProps<label.Key> {
  isCreate?: boolean;
  visible?: boolean;
  onClose?: () => void;
}

const LabelListItem = ({
  isCreate = false,
  onClose,
  visible = true,
  ...rest
}: LabelListItemProps) => {
  const { itemKey } = rest;
  const initialValues = List.useItem<string, label.Label>(itemKey);
  const { form, save } = Label.useForm({
    query: {},
    initialValues,
    autoSave: !isCreate,
    afterSave: useCallback(
      ({ reset }: Flux.AfterSaveParams<query.Params, typeof Label.formSchema>) => {
        onClose?.();
        if (isCreate) reset({ name: "", color: color.construct("#000000") });
      },
      [isCreate, onClose],
    ),
    sync: true,
  });
  const inputRef = useRef<HTMLInputElement>(null);
  const { update: handleDelete } = Label.useDelete();
  useEffect(() => {
    if (isCreate && visible) inputRef.current?.focus();
  }, [isCreate, visible]);
  const ref = useRef<HTMLDivElement>(null);
  useClickOutside({
    ref,
    onClickOutside: useCallback(() => {
      if (!isCreate) return;
      if (form.validate()) save();
      else onClose?.();
    }, [isCreate, form, save, onClose]),
  });
  return (
    <List.Item
      ref={ref}
      className={CSS(
        CSS.BE("label", "list-item"),
        isCreate && CSS.M("create"),
        PCSS.visible(visible),
      )}
      align="center"
      justify="between"
      {...rest}
    >
      <Flex.Box x gap="small" align="center">
        <Form.Form<typeof Label.formSchema> {...form}>
          <Form.Field<string>
            hideIfNull
            path="color"
            padHelpText={false}
            showLabel={false}
          >
            {({ onChange, preview: _, ...p }) => (
              <Color.Swatch onChange={(v) => onChange(color.hex(v))} {...p} />
            )}
          </Form.Field>
          <Form.TextField
            showLabel={false}
            hideIfNull
            path="name"
            showHelpText={false}
            padHelpText={false}
            inputProps={{
              ref: inputRef,
              placeholder: "Label Name",
              variant: "text",
              selectOnFocus: true,
              autoFocus: isCreate,
              resetOnBlurIfEmpty: true,
              onlyChangeOnBlur: !isCreate,
            }}
          />
        </Form.Form>
      </Flex.Box>
      {isCreate ? (
        <Flex.Box pack>
          <Button.Button
            variant="filled"
            size="small"
            onClick={() => save()}
            trigger={visible ? ["Enter"] : undefined}
          >
            <Icon.Check />
          </Button.Button>
          <Button.Button variant="outlined" size="small" onClick={onClose}>
            <Icon.Close />
          </Button.Button>
        </Flex.Box>
      ) : (
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

const listItem = Component.renderProp(LabelListItem);

export const useEditModal = Modals.create(() => {
  const { data, getItem, retrieve, subscribe } = Label.useList();
  const { fetchMore, search } = List.usePager({ retrieve, pageSize: 15 });
  const [newFormVisible, setNewFormVisible] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  return (
    <Modals.Frame y className={CSS.BE("label", "edit")}>
      <Modals.Header icon={<Icon.Label />}>Labels.Edit</Modals.Header>
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
          <LabelListItem
            key="form"
            index={0}
            itemKey=""
            isCreate
            visible={newFormVisible}
            onClose={() => setNewFormVisible(false)}
          />
          <List.Items
            grow
            emptyContent={
              !newFormVisible && (
                <Flex.Box center>
                  <Text.Text level="h4" color={9}>
                    No labels created
                  </Text.Text>
                </Flex.Box>
              )
            }
          >
            {listItem}
          </List.Items>
          {!newFormVisible && (
            <PlatformButton.Create
              onClick={() => setNewFormVisible(true)}
              className={CSS.BE("label", "create")}
            >
              New Label
            </PlatformButton.Create>
          )}
        </Flex.Box>
      </List.Frame>
    </Modals.Frame>
  );
});
