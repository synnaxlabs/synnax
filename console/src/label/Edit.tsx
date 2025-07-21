// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/label/Edit.css";

import { label } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Color,
  Component,
  Form,
  Icon,
  List,
  Text,
} from "@synnaxlabs/pluto";
import { type change, color, uuid } from "@synnaxlabs/x";
import { z } from "zod";

import { CSS } from "@/css";
import { Layout } from "@/layout";

const formSchema = z.object({
  labels: label.newZ.array(),
});

const LabelListItem = (props: List.ItemProps<label.Key>) => {
  const { itemKey } = props;
  const { remove } = Form.fieldListUtils(Form.useContext(), "labels");
  return (
    <List.Item
      highlightHovered={false}
      className={CSS.BE("label", "list-item")}
      allowSelect={false}
      align="center"
      style={{ padding: "2rem 4rem" }}
      justify="spaceBetween"
      {...props}
    >
      <Align.Space x size="small">
        <Form.Field<string>
          hideIfNull
          path={`labels.${itemKey}.color`}
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
          path={`labels.${itemKey}.name`}
          padHelpText={false}
          inputProps={{
            placeholder: "Label Name",
            variant: "shadow",
            selectOnFocus: true,
            resetOnBlurIfEmpty: true,
            onlyChangeOnBlur: true,
          }}
        />
      </Align.Space>
      <Button.Icon onClick={() => remove(itemKey)} style={{ width: "fit-content" }}>
        <Icon.Delete />
      </Button.Icon>
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
  window: { navTop: true, size: { height: 800, width: 500 } },
};

const listItem = Component.renderProp(LabelListItem);

const initialState = formSchema.parse({ labels: [] });

export const Edit: Layout.Renderer = () => {
  const ctx = Form.useSynced<typeof formSchema, change.Change<string, label.Label>[]>({
    values: initialState,
    key: ["labels"],
    name: "Labels",
    queryFn: async ({ client }) => ({ labels: await client.labels.page(0, 100) }),
    applyChanges: async ({ values, path, prev, client }) => {
      if (path === "labels") {
        const tPrev = prev as label.Label[];
        if (values.labels.length >= tPrev.length) return;
        const newKeys = values.labels.map((l) => l.key);
        const oldKeys = tPrev.map((l) => l.key);
        const key = oldKeys.find((k) => !newKeys.includes(k));
        if (key == null) return;
        await client.labels.delete(key);
        return;
      }
      const idx = Number(path.split(".")[1]);
      const label = values.labels[idx];
      if (label == null) return;
      await client.labels.create({ ...label, color: color.hex(label.color) });
    },
  });

  const { data, push } = Form.useFieldList<label.Key, label.Label, typeof formSchema>(
    "labels",
    { ctx },
  );
  const theme = Layout.useSelectTheme();

  return (
    <Align.Space y style={{ padding: "2rem" }} grow>
      <Form.Form<typeof formSchema> {...ctx}>
        <List.Frame<label.Key, label.Label> data={data}>
          <Align.Space x justify="spaceBetween">
            <Button.Button
              onClick={() => {
                const newColors = theme?.colors.visualization.palettes.default ?? [];
                const v = color.hex(newColors[data.length % newColors.length]);
                push({
                  key: uuid.create(),
                  name: "New Label",
                  color: v,
                });
              }}
              startIcon={<Icon.Add />}
              style={{ width: "fit-content" }}
              iconSpacing="small"
            >
              Add Label
            </Button.Button>
          </Align.Space>
          <List.Items
            style={{
              borderRadius: "1rem",
              border: "var(--pluto-border)",
              maxHeight: "calc(100% - 10rem)",
            }}
            emptyContent={
              <Align.Center>
                <Text.Text level="h3" shade={10}>
                  No labels created
                </Text.Text>
              </Align.Center>
            }
          >
            {listItem}
          </List.Items>
        </List.Frame>
      </Form.Form>
    </Align.Space>
  );
};
