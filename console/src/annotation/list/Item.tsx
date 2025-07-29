// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  annotation,
  type ontology,
  TimeRange,
  type TimeStamp,
} from "@synnaxlabs/client";
import {
  Align,
  Annotation,
  Button,
  Dialog,
  Form,
  Icon,
  List,
  Menu,
  Ranger,
  Text,
  User as PUser,
} from "@synnaxlabs/pluto";
import { useCallback, useMemo, useRef, useState } from "react";

import { ContextMenu } from "@/annotation/list/ContextMenu";
import { CSS } from "@/css";
import { useConfirmDelete } from "@/ontology/hooks";
import { Triggers } from "@/triggers";
import { User } from "@/user";

export interface AnnotationListItemProps extends List.ItemProps<annotation.Key> {
  parent?: ontology.ID;
  isCreate?: boolean;
  parentStart?: TimeStamp;
}

// Hash a string into a deterministic 32-bit integer
const stringToHash = (str: string): number => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
    hash |= 0; // Convert to 32-bit integer
  }
  return Math.abs(hash);
};

// Convert a hash into an HSL color
const hashToHSL = (hash: number, offset: number = 0): string => {
  const hue = (hash + offset) % 360;
  const saturation = 60 + (hash % 30); // Range: 60–89%
  const lightness = 50 + (hash % 10); // Range: 50–59%
  return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
};

// Generate a consistent linear gradient for a username
export const usernameToGradient = (username: string): string => {
  const baseHash = stringToHash(username);
  const color1 = hashToHSL(baseHash, 0);
  const color2 = hashToHSL(baseHash, 120);
  const color3 = hashToHSL(baseHash, 240);
  return `linear-gradient(135deg, ${color1}, ${color2}, ${color3})`;
};

export const Item = ({
  parent,
  parentStart,
  isCreate,
  ...rest
}: AnnotationListItemProps) => {
  const { itemKey } = rest;
  const initialValues = List.useItem<string, annotation.Annotation>(itemKey);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const [edit, setEdit] = useState(isCreate);
  const [focused, setFocused] = useState(false);
  const currentUser = User.useSelect();
  const values = useMemo(
    () => ({
      parent,
      key: itemKey.length > 0 ? itemKey : undefined,
      message: initialValues?.message ?? "",
      timeRange: initialValues?.timeRange.numeric ?? TimeRange.ZERO.numeric,
    }),
    [initialValues],
  );
  const { form, save } = Annotation.useForm({
    params: {},
    initialValues: values,
    sync: !isCreate,
    afterSave: ({ form }) => {
      if (isCreate) form.reset();
      else setEdit(false);
    },
  });
  const { data: creator } = PUser.retrieveCreator.useDirect({
    params: { id: annotation.ontologyID(itemKey) },
  });

  const menuProps = Menu.useContextMenu();

  const startEditing = useCallback(() => {
    setEdit(true);
    setTimeout(() => {
      inputRef.current?.focus();
      inputRef.current?.setSelectionRange(
        inputRef.current?.value.length ?? 0,
        inputRef.current?.value.length ?? 0,
      );
    }, 0);
  }, []);

  const confirmDelete = useConfirmDelete({
    type: "Annotation",
  });

  const { update: del } = Annotation.useDelete({ params: { key: itemKey } });

  const handleDelete = useCallback(() => {
    confirmDelete({
      name: "This annotation",
    })
      .then((confirmed) => confirmed && del())
      .catch(console.error);
  }, [confirmDelete, del]);

  return (
    <List.Item
      {...rest}
      bordered
      variant="outlined"
      borderShade={6}
      onContextMenu={menuProps.open}
      y
      className={CSS.BE("annotation", "list-item")}
      gap="small"
    >
      <Menu.ContextMenu
        menu={(p) => (
          <ContextMenu {...p} onEdit={startEditing} onDelete={handleDelete} />
        )}
        {...menuProps}
      />
      <Align.Space x grow justify="spaceBetween" align="center">
        <Align.Space x align="center" gap="small">
          <div
            className={CSS.BE("annotation", "list-item__avatar")}
            style={{ background: usernameToGradient("synnax") }}
          />
          <Text.Text level="h5" shade={9} weight={450}>
            {creator?.username ?? currentUser.username}
          </Text.Text>
        </Align.Space>
        <Align.Space x gap="small">
          {initialValues?.timeRange && (
            <Ranger.TimeRangeChip
              level="small"
              timeRange={initialValues.timeRange}
              collapseZero
              offsetFrom={parentStart}
              showAgo
              variant="outlined"
            />
          )}
          {!edit && (
            <Dialog.Frame variant="floating" location={{ x: "right", y: "bottom" }}>
              <Dialog.Trigger iconOnly hideCaret startIcon={<Icon.KebabMenu />} />
              <Dialog.Dialog bordered borderShade={5} rounded={1}>
                <Align.Space gap="tiny" background={1} style={{ padding: "1rem" }}>
                  <ContextMenu onEdit={startEditing} onDelete={handleDelete} />
                </Align.Space>
              </Dialog.Dialog>
            </Dialog.Frame>
          )}
        </Align.Space>
      </Align.Space>
      <Form.Form<typeof Annotation.formSchema> {...form}>
        {edit ? (
          <Form.TextAreaField
            path="message"
            showLabel={false}
            showHelpText={false}
            inputProps={{
              placeholder: "Leave a comment...",
              level: "h5",
              ref: inputRef,
              onFocus: () => setFocused(true),
              onBlur: () => setFocused(false),
            }}
          />
        ) : (
          <Text.Text level="h5" shade={11} weight={450}>
            {initialValues?.message}
          </Text.Text>
        )}
      </Form.Form>
      {edit && (
        <Align.Space x grow justify="spaceBetween">
          <Align.Space x align="center">
            <Triggers.SaveHelpText noBar />
          </Align.Space>
          <Button.Icon
            variant="outlined"
            shade={2}
            onClick={() => save()}
            triggers={focused ? Triggers.SAVE : undefined}
          >
            <Icon.Arrow.Up />
          </Button.Icon>
        </Align.Space>
      )}
    </List.Item>
  );
};
