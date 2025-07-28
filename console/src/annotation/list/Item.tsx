// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/annotation/list/Item.css";

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
  Form,
  Icon,
  List,
  Ranger,
  Text,
  User as PUser,
} from "@synnaxlabs/pluto";
import { useMemo, useState } from "react";

import { CSS } from "@/css";
import { Triggers } from "@/triggers";

export interface AnnotationListItemProps extends List.ItemProps<annotation.Key> {
  parent?: ontology.ID;
  isCreate?: boolean;
  parentStart?: TimeStamp;
}

export const ListItem = ({
  parent,
  parentStart,
  isCreate,
  ...rest
}: AnnotationListItemProps) => {
  const { itemKey } = rest;
  const initialValues = List.useItem<string, annotation.Annotation>(itemKey);
  const [edit, setEdit] = useState(isCreate);
  const [focused, setFocused] = useState(false);
  const values = useMemo(
    () => ({
      key: itemKey.length > 0 ? itemKey : undefined,
      message: initialValues?.message ?? "",
      timeRange: initialValues?.timeRange.numeric ?? TimeRange.ZERO.numeric,
    }),
    [initialValues],
  );
  const { form, save } = Annotation.useForm({
    params: { parent },
    initialValues: values,
    sync: !isCreate,
    afterSave: ({ form }) => {
      if (isCreate) form.reset();
    },
  });
  const { data: creator } = PUser.retrieveCreator.useDirect({
    params: { id: annotation.ontologyID(itemKey) },
  });

  return (
    <List.Item
      {...rest}
      bordered
      variant="outlined"
      borderShade={6}
      y
      className={CSS.BE("annotation", "list-item")}
      gap="small"
    >
      <Align.Space x grow justify="spaceBetween" align="center">
        <Align.Space x align="center" gap="small">
          <div className={CSS.BE("annotation", "list-item__avatar")} />
          <Text.Text level="h5" shade={9} weight={450}>
            {creator?.username}
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
            <Button.Icon variant="text" shade={1} size="small">
              <Icon.KebabMenu />
            </Button.Icon>
          )}
        </Align.Space>
      </Align.Space>
      <Form.Form<typeof Annotation.formSchema> {...form}>
        {edit ? (
          <Form.TextAreaField
            path="message"
            showLabel={false}
            inputProps={{
              placeholder: "Leave a comment...",
              level: "h5",
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
