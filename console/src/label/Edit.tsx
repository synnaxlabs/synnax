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
  Align,
  Button,
  Color,
  Component,
  CSS as PCSS,
  Divider,
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
    params: {},
    initialValues,
    autoSave: !isCreate,
    afterSave: useCallback(
      ({ form }: Flux.AfterSaveArgs<Flux.Params, typeof Label.formSchema>) => {
        onClose?.();
        if (isCreate)
          form.reset({
            name: "",
            color: "#000000",
          });
      },
      [isCreate, onClose],
    ),
    sync: true,
  });
  const inputRef = useRef<HTMLInputElement>(null);
  const { update: handleDelete } = Label.useDelete({ params: { key: itemKey } });
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
      justify="spaceBetween"
      {...rest}
    >
      <Align.Space x gap="small" align="center">
        <Form.Form<typeof Label.formSchema> {...form}>
          <Form.Field<string>
            hideIfNull
            path={`color`}
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
            path={`name`}
            showHelpText={false}
            padHelpText={false}
            inputProps={{
              ref: inputRef,
              placeholder: "Label Name",
              variant: "shadow",
              selectOnFocus: true,
              autoFocus: isCreate,
              resetOnBlurIfEmpty: true,
              onlyChangeOnBlur: !isCreate,
            }}
          />
        </Form.Form>
      </Align.Space>
      {isCreate ? (
        <Align.Pack>
          <Button.Icon
            variant="filled"
            size="small"
            onClick={() => save()}
            triggers={visible ? [["Enter"]] : undefined}
          >
            <Icon.Check />
          </Button.Icon>
          <Button.Icon variant="outlined" size="small" onClick={onClose}>
            <Icon.Close />
          </Button.Icon>
        </Align.Pack>
      ) : (
        <Button.Icon
          className={CSS.BE("label", "delete")}
          variant="outlined"
          size="small"
          onClick={() => handleDelete()}
        >
          <Icon.Delete />
        </Button.Icon>
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
  const { fetchMore, search } = List.usePager({ retrieve });
  const [newFormVisible, setNewFormVisible] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  return (
    <Align.Space y grow empty>
      <List.Frame<label.Key, label.Label>
        data={data}
        getItem={getItem}
        onFetchMore={fetchMore}
        subscribe={subscribe}
      >
        <Align.Space x justify="spaceBetween" style={{ padding: "2rem" }}>
          <Input.Text
            placeholder={
              <Text.WithIcon level="p" startIcon={<Icon.Search />}>
                Search Labels
              </Text.WithIcon>
            }
            value={searchTerm}
            onChange={(v) => {
              setSearchTerm(v);
              search(v);
            }}
          />
          <Button.Button
            variant="filled"
            startIcon={<Icon.Add />}
            style={{ width: "fit-content" }}
            gap="small"
            onClick={() => setNewFormVisible(true)}
          >
            Add Label
          </Button.Button>
        </Align.Space>
        <Divider.Divider x />
        <Align.Space
          y
          style={{
            borderRadius: "1rem",
            height: "100%",
          }}
          empty
        >
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
                <Align.Center>
                  <Text.Text level="h4" shade={8}>
                    No labels created
                  </Text.Text>
                </Align.Center>
              )
            }
          >
            {listItem}
          </List.Items>
        </Align.Space>
      </List.Frame>
    </Align.Space>
  );
};
