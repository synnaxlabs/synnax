// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/label/Edit.css";

import { label } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Color,
  componentRenderProp,
  Form,
  Input,
  List,
  Text,
} from "@synnaxlabs/pluto";
import { type change } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { Layout } from "@/layout";

const formSchema = z.object({
  labels: label.newZ.array(),
});

const LabelListItem = (props: List.ItemProps<string, label.Label>) => {
  const { index } = props;
  const utils = Form.fieldArrayUtils(Form.useContext(), "labels");
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
      <Align.Space direction="x" size="small">
        <Form.Field<string>
          hideIfNull
          path={`labels.${index}.color`}
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
          path={`labels.${index}.name`}
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
      <Button.Icon onClick={() => utils.remove(index)} style={{ width: "fit-content" }}>
        <Icon.Delete />
      </Button.Icon>
    </List.ItemFrame>
  );
};

export const EDIT_LAYOUT_TYPE = "editLabels";

export const createEditLayout = (): Layout.State => ({
  key: EDIT_LAYOUT_TYPE,
  type: EDIT_LAYOUT_TYPE,
  windowKey: EDIT_LAYOUT_TYPE,
  name: "Edit Labels",
  location: "modal",
  icon: "Label",
  window: {
    size: { height: 800, width: 500 },
    navTop: true,
  },
});

const listItem = componentRenderProp(LabelListItem);

const initialState = formSchema.parse({ labels: [] });

export const Edit: Layout.Renderer = (): ReactElement => {
  const methods = Form.useSynced<
    typeof formSchema,
    change.Change<string, label.Label>[]
  >({
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
      await client.labels.create({ ...label, color: Color.toHex(label.color) });
    },
  });

  const arr = Form.useFieldArray<label.Label>({ path: "labels", ctx: methods });
  const theme = Layout.useSelectTheme();

  return (
    <Align.Space direction="y" style={{ padding: "2rem" }} grow>
      <Form.Form {...methods}>
        <List.List
          data={arr.value}
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
              onClick={() => {
                const newColors = theme?.colors.visualization.palettes.default ?? [];
                const color = newColors[arr.value.length % newColors.length].hex;
                arr.push({
                  key: uuid(),
                  name: "New Label",
                  color,
                });
              }}
              startIcon={<Icon.Add />}
              style={{ width: "fit-content" }}
              iconSpacing="small"
            >
              Add Label
            </Button.Button>
          </Align.Space>
          <List.Core
            style={{
              borderRadius: "1rem",
              border: "var(--pluto-border)",
              maxHeight: "calc(100% - 10rem)",
            }}
            grow
          >
            {listItem}
          </List.Core>
        </List.List>
      </Form.Form>
    </Align.Space>
  );
};
