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
  Annotation,
  Button,
  Dialog,
  Flex,
  type Flux,
  Form,
  Icon,
  List,
  Menu,
  Telem,
  Text,
  User as PUser,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
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

export const Item = ({
  parent,
  parentStart,
  isCreate,
  ...rest
}: AnnotationListItemProps) => {
  const { itemKey } = rest;
  const initialValues = List.useItem<string, annotation.Annotation>(itemKey);
  const inputRef = useRef<HTMLInputElement>(null);
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
    query: {},
    initialValues: values,
    sync: !isCreate,
    afterSave: ({ reset }) => {
      if (isCreate) reset();
      else setEdit(false);
    },
  });
  const otgID = annotation.ontologyID(itemKey);
  const { data: creator } = PUser.useRetrieveCreator({ id: otgID });

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

  const { update: del } = Annotation.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<PUser.DeleteParams>) => {
        const confirmed = await confirmDelete({
          name: "This annotation",
        });
        if (!confirmed) return false;
        return data;
      },
      [],
    ),
  });
  const handleDelete = useCallback(() => del(itemKey), [itemKey, del]);

  const username = creator?.username ?? currentUser.username;

  return (
    <List.Item
      {...rest}
      bordered
      variant="outlined"
      borderColor={6}
      onContextMenu={menuProps.open}
      y
      className={CSS.BE("annotation", "list-item")}
    >
      <Menu.ContextMenu
        menu={(p) => (
          <ContextMenu {...p} onEdit={startEditing} onDelete={handleDelete} />
        )}
        {...menuProps}
      />
      <Flex.Box x grow justify="between" align="center">
        <Flex.Box x align="center" gap="small">
          <div
            className={CSS.BE("annotation", "list-item__avatar")}
            style={{ background: PUser.avatar(username) }}
          />
          <Text.Text level="h5" color={9} weight={450}>
            {username}
          </Text.Text>
        </Flex.Box>
        <Flex.Box x gap="small">
          {initialValues?.timeRange && (
            <Telem.Text.TimeRange
              level="small"
              // collapseZero
              // offsetFrom={parentStart}
              // showAgo
              // variant="outlined"
            >
              {initialValues.timeRange}
            </Telem.Text.TimeRange>
          )}
          {!edit && (
            <Dialog.Frame
              variant="floating"
              location={{
                targetCorner: location.BOTTOM_RIGHT,
                dialogCorner: location.TOP_RIGHT,
              }}
            >
              <Dialog.Trigger hideCaret variant="text" contrast={1}>
                <Icon.KebabMenu />
              </Dialog.Trigger>
              <Dialog.Dialog bordered borderColor={5} rounded={1}>
                <Flex.Box gap="tiny" background={1} style={{ padding: "1rem" }}>
                  <ContextMenu onEdit={startEditing} onDelete={handleDelete} />
                </Flex.Box>
              </Dialog.Dialog>
            </Dialog.Frame>
          )}
        </Flex.Box>
      </Flex.Box>
      <Form.Form<typeof Annotation.formSchema> {...form}>
        {edit ? (
          <Form.TextField
            path="message"
            showLabel={false}
            showHelpText={false}
            inputProps={{
              placeholder: "Leave a comment...",
              variant: "text",
              area: true,
              level: "h5",
              ref: inputRef,
              onFocus: () => setFocused(true),
              onBlur: () => setFocused(false),
            }}
          />
        ) : (
          <Text.Text level="h5" color={11} weight={350}>
            {initialValues?.message}
          </Text.Text>
        )}
      </Form.Form>
      {edit && (
        <Flex.Box x grow justify="between">
          <Flex.Box x align="center">
            <Triggers.SaveHelpText noBar />
          </Flex.Box>
          <Button.Button
            variant="outlined"
            contrast={2}
            onClick={() => save()}
            trigger={focused ? Triggers.SAVE : undefined}
          >
            <Icon.Arrow.Up />
          </Button.Button>
        </Flex.Box>
      )}
    </List.Item>
  );
};
