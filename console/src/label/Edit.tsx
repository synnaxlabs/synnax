// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/label/Edit.css";

import { type label } from "@synnaxlabs/client";
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

import { CSS } from "@/css";
import { type Layout } from "@/layout";

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
      ({
        reset,
      }: Flux.AfterSaveParams<
        Flux.Shape,
        typeof Label.formSchema,
        Label.FluxSubStore
      >) => {
        onClose?.();
        if (isCreate) reset({ name: "", color: "#000000" });
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
      highlightHovered={false}
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
            {({ onChange, variant: _, ...p }) => (
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
          onClick={() => handleDelete(itemKey)}
          className={CSS.BE("label", "delete")}
        >
          <Icon.Delete />
        </Button.Button>
      )}
    </List.Item>
  );
};

export const EDIT_LAYOUT_TYPE = "editLabels";

export const EDIT_LAYOUT: Layout.BaseState = {
  key: EDIT_LAYOUT_TYPE,
  type: EDIT_LAYOUT_TYPE,
  name: "Labels.Edit",
  location: "modal",
  icon: "Label",
  window: { navTop: true, size: { height: 700, width: 450 } },
};

const listItem = Component.renderProp(LabelListItem);

export const Edit: Layout.Renderer = () => {
  const { data, getItem, retrieve, subscribe } = Label.useList();
  const { fetchMore, search } = List.usePager({ retrieve, pageSize: 15 });
  const [newFormVisible, setNewFormVisible] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  return (
    <Flex.Box y grow empty className={CSS.BE("label", "edit")}>
      <List.Frame<label.Key, label.Label>
        data={data}
        getItem={getItem}
        onFetchMore={fetchMore}
        subscribe={subscribe}
      >
        <Flex.Box x justify="between" className={CSS.BE("label", "edit-header")}>
          <Input.Text
            placeholder={
              <>
                <Icon.Search />
                Search labels
              </>
            }
            value={searchTerm}
            onChange={(v) => {
              setSearchTerm(v);
              search(v);
            }}
          />
          <Button.Button
            variant="filled"
            className={CSS.BE("label", "add-btn")}
            gap="small"
            onClick={() => setNewFormVisible(true)}
          >
            <Icon.Add />
          </Button.Button>
        </Flex.Box>
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
                  <Text.Text level="h4" color={8}>
                    No labels created
                  </Text.Text>
                </Flex.Box>
              )
            }
          >
            {listItem}
          </List.Items>
        </Flex.Box>
      </List.Frame>
    </Flex.Box>
  );
};
