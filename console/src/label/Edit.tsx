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
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Color,
  componentRenderProp,
  Form,
  Input,
  Label,
  List,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { useState } from "react";
import { v4 as uuidv4 } from "uuid";

import { CSS } from "@/css";
import { type Layout } from "@/layout";

interface LabelFormProps {
  values: label.Label;
  isNew: boolean;
  close?: () => void;
}

const LabelForm = ({ values, isNew, close }: LabelFormProps) => {
  const { save, ...methods } = Label.useSyncedForm({
    key: values.key,
    values,
    autoSave: !isNew,
  });
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return (
    <Form.Form {...methods}>
      <Align.Space direction="x" size="small">
        <Form.Field<string>
          hideIfNull
          path={`color`}
          padHelpText={false}
          showLabel={false}
        >
          {({ onChange, variant: _, ...p }) => (
            <Color.Swatch onChange={(color) => onChange(color.hex)} {...p} />
          )}
        </Form.Field>
        <Form.TextField
          showLabel={false}
          hideIfNull
          path={`name`}
          padHelpText={false}
          inputProps={{
            placeholder: "Label Name",
            variant: "shadow",
            selectOnFocus: true,
            resetOnBlurIfEmpty: true,
            onlyChangeOnBlur: true,
          }}
        />
        {isNew ? (
          <Align.Space direction="x">
            <Button.Icon
              onClick={() => {
                save();
                close?.();
              }}
            >
              <Icon.Check />
            </Button.Icon>
            <Button.Icon onClick={close}>
              <Icon.Close />
            </Button.Icon>
          </Align.Space>
        ) : (
          <Button.Icon
            onClick={() => {
              client?.labels.delete(values.key).catch(handleError);
            }}
          >
            <Icon.Delete />
          </Button.Icon>
        )}
      </Align.Space>
    </Form.Form>
  );
};

const LabelListItem = (props: List.ItemProps<string, label.Label>) => {
  const { entry } = props;
  return (
    <List.ItemFrame
      highlightHovered={false}
      className={CSS.BE("label", "list-item")}
      allowSelect={false}
      align="center"
      style={{ padding: "2rem 4rem" }}
      justify="spaceBetween"
      {...props}
    >
      <LabelForm key={entry.key} values={entry} isNew={false} />
    </List.ItemFrame>
  );
};

export const EDIT_LAYOUT_TYPE = "editLabels";

export const EDIT_LAYOUT: Layout.BaseState = {
  key: EDIT_LAYOUT_TYPE,
  type: EDIT_LAYOUT_TYPE,
  name: "Labels.Edit",
  location: "modal",
  icon: "Label",
  window: { navTop: true, size: { height: 800, width: 500 } },
};

const listItem = componentRenderProp(LabelListItem);

export const Edit: Layout.Renderer = () => {
  const labels = Label.useRetrieveMany();
  const [showNew, setShowNew] = useState(false);
  return (
    <Align.Space direction="y" style={{ padding: "2rem" }} grow>
      <List.List
        data={labels.value}
        emptyContent={
          <Align.Center>
            <Text.Text level="h3" shade={6}>
              No labels created
            </Text.Text>
          </Align.Center>
        }
      >
        <Align.Space direction="x" justify="spaceBetween">
          <List.Filter>
            {(p) => (
              <Input.Text
                {...p}
                placeholder="Search Labels"
                style={{ width: "unset" }}
              />
            )}
          </List.Filter>
          <Button.Button
            startIcon={<Icon.Add />}
            style={{ width: "fit-content" }}
            iconSpacing="small"
            onClick={() => setShowNew(true)}
          >
            New Label
          </Button.Button>
        </Align.Space>
        {showNew && (
          <LabelForm
            values={{
              key: uuidv4(),
              name: "",
              color: "#000000",
            }}
            isNew={true}
            close={() => setShowNew(false)}
          />
        )}
        <List.Core>{listItem}</List.Core>
      </List.List>
    </Align.Space>
  );
};
