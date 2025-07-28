// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Channel,
  Form,
  Input,
  Nav,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { useState } from "react";

import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const CREATE_LAYOUT_TYPE = "createChannel";

export const CREATE_LAYOUT: Layout.BaseState = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  name: "Channel.Create",
  icon: "Channel",
  location: "modal",
  window: {
    resizable: false,
    size: { height: 375, width: 700 },
    navTop: true,
    showTitle: true,
  },
};

export const Create: Layout.Renderer = ({ onClose }) => {
  const [createMore, setCreateMore] = useState(false);
  const { form, variant, save } = Channel.useForm({
    params: {},
    afterSave: ({ form }) => {
      if (createMore) form.reset();
      else onClose();
    },
  });

  const isIndex = Form.useFieldValue<boolean, boolean, typeof Channel.formSchema>(
    "isIndex",
    {
      ctx: form,
    },
  );
  const isVirtual = Form.useFieldValue<boolean, boolean, typeof Channel.formSchema>(
    "virtual",
    {
      ctx: form,
    },
  );

  return (
    <Align.Space className={CSS.B("channel-edit-layout")} grow empty>
      <Align.Space className="console-form" style={{ padding: "3rem" }} grow>
        <Form.Form<typeof Channel.formSchema> {...form}>
          <Form.Field<string> path="name" label="Name">
            {(p) => (
              <Input.Text
                autoFocus
                level="h2"
                variant="natural"
                placeholder="Name"
                {...p}
              />
            )}
          </Form.Field>
          <Align.Space x gap="large">
            <Form.SwitchField
              path="virtual"
              label="Virtual"
              inputProps={{ disabled: isIndex }}
              onChange={(v, ctx) => {
                if (!v) {
                  const dataType = ctx.get<string>("dataType").value;
                  if (new DataType(dataType).isVariable)
                    ctx.set("dataType", DataType.FLOAT32.toString());
                  return;
                }
                ctx.set("isIndex", false);
                ctx.set("index", 0);
              }}
            />
            <Form.SwitchField
              path="isIndex"
              label="Is Index"
              inputProps={{ disabled: isVirtual }}
              onChange={(v, ctx) => {
                if (!v) return;
                ctx.set("dataType", DataType.TIMESTAMP.toString());
                if (ctx.get("index").value !== 0) ctx.set("index", 0);
              }}
            />
            <Form.Field<string> path="dataType" label="Data Type" grow>
              {({ variant: _, ...p }) => (
                <Telem.SelectDataType
                  {...p}
                  disabled={isIndex}
                  zIndex={100}
                  hideVariableDensity={!isVirtual}
                />
              )}
            </Form.Field>
          </Align.Space>
          <Form.Field<channel.Key> path="index" label="Index">
            {({ value, onChange }) => (
              <Channel.SelectSingle
                value={value}
                onChange={onChange}
                initialParams={{ isIndex: true }}
                disabled={isIndex || isVirtual}
                allowNone={false}
                zIndex={100}
              />
            )}
          </Form.Field>
        </Form.Form>
      </Align.Space>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText />
        <Nav.Bar.End align="center" gap="large">
          <Align.Space x align="center" gap="small">
            <Input.Switch value={createMore} onChange={setCreateMore} />
            <Text.Text level="p" shade={11}>
              Create More
            </Text.Text>
          </Align.Space>
          <Button.Button
            disabled={variant === "loading"}
            loading={variant === "loading"}
            variant="filled"
            onClick={() => save()}
            triggers={[Triggers.SAVE]}
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Align.Space>
  );
};
