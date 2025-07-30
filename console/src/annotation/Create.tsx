// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/annotation/Create.css";

import { type ontology, ranger, TimeStamp } from "@synnaxlabs/client";
import {
  Align,
  Annotation,
  Button,
  Form,
  Icon,
  Input,
  Nav,
  Ranger,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useRef } from "react";
import { type z } from "zod";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export interface CreateLayoutArgs
  extends Partial<z.infer<typeof Annotation.formSchema>> {}

export const CREATE_LAYOUT_TYPE = "editAnnotation";

export const CREATE_LAYOUT: Layout.BaseState<CreateLayoutArgs> = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  location: "modal",
  name: "Annotation.Create",
  icon: "Annotation",
  window: {
    resizable: false,
    size: { height: 420, width: 650 },
    navTop: true,
  },
};

export const createCreateLayout = (
  initial: CreateLayoutArgs = {},
): Layout.BaseState<CreateLayoutArgs> => ({
  ...CREATE_LAYOUT,
  args: initial,
});

export const Create: Layout.Renderer = (props) => {
  const { layoutKey, onClose } = props;
  const now = useRef(Number(TimeStamp.now().valueOf())).current;
  const args = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);

  const client = Synnax.use();
  const clientExists = client != null;

  const { form, save, variant } = Annotation.useForm({
    params: { key: args?.key },
    autoSave: false,
    initialValues: {
      key: uuid.create(),
      message: "",
      timeRange: { start: now, end: now },
      parent: args?.parent,
      ...args,
    },
    afterSave: () => onClose(),
  });

  return (
    <Align.Space className={CSS.B("annotation-create-layout")} grow empty>
      <Align.Space
        className="console-form"
        justify="center"
        style={{ padding: "1.5rem 2.5rem" }}
        grow
      >
        <Form.Form<typeof Annotation.formSchema> {...form}>
          <Align.Space direction="y" gap="small">
            <Form.Field<string> path="message">
              {(p) => (
                <Input.TextArea
                  placeholder="Enter annotation message..."
                  style={{ minHeight: "100px" }}
                  autoFocus
                  {...p}
                />
              )}
            </Form.Field>
            <Form.Field<ontology.ID> path="parent" label="Parent Range">
              {({ onChange, value }) => (
                <Ranger.SelectSingle
                  style={{ width: "100%" }}
                  value={value?.key}
                  onChange={(v: ranger.Key) => {
                    if (v == null) return;
                    onChange(ranger.ontologyID(v));
                  }}
                  allowNone={false}
                />
              )}
            </Form.Field>
            <Align.Space direction="x" gap="large" align="center">
              <Form.Field<number> path="timeRange.start" label="From" required={false}>
                {(p) => <Input.DateTime level="p" variant="natural" {...p} />}
              </Form.Field>
              <Text.WithIcon level="p" startIcon={<Icon.Arrow.Right />} />
              <Form.Field<number> path="timeRange.end" label="To" required={false}>
                {(p) => <Input.DateTime level="p" variant="natural" {...p} />}
              </Form.Field>
            </Align.Space>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action="Save Annotation" />
        <Nav.Bar.End>
          <Button.Button
            variant="filled"
            onClick={() => save()}
            disabled={!clientExists || variant === "loading"}
            tooltip={clientExists ? "Save Annotation" : "No Cluster Connected"}
            tooltipLocation="bottom"
            loading={variant === "loading"}
            triggers={Triggers.SAVE}
          >
            Save Annotation
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Align.Space>
  );
};
