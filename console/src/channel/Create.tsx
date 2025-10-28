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
  Button,
  Channel,
  Flex,
  Form,
  Input,
  Nav,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { useState } from "react";

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

const INDEX_QUERY: Partial<Channel.RetrieveMultipleQuery> = { isIndex: true };

export const Create: Layout.Renderer = ({ onClose }) => {
  const [createMore, setCreateMore] = useState(false);
  const { form, variant, save } = Channel.useForm({
    query: {},
    afterSave: ({ reset }) => {
      if (createMore) reset(Channel.ZERO_FORM_VALUES);
      else onClose();
    },
  });
  const isIndex = Form.useFieldValue<boolean, boolean, typeof Channel.formSchema>(
    "isIndex",
    { ctx: form },
  );
  const isVirtual = Form.useFieldValue<boolean, boolean, typeof Channel.formSchema>(
    "virtual",
    { ctx: form },
  );
  return (
    <Flex.Box grow empty>
      <Flex.Box className="console-form" style={{ padding: "3rem" }} grow>
        <Form.Form<typeof Channel.formSchema> {...form}>
          <Form.Field<string> path="name" label="Name">
            {(p) => (
              <Input.Text
                autoFocus
                level="h2"
                variant="text"
                placeholder="Name"
                {...p}
              />
            )}
          </Form.Field>
          <Flex.Box x gap="large">
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
                  full="x"
                />
              )}
            </Form.Field>
          </Flex.Box>
          <Form.Field<channel.Key> path="index" label="Index">
            {({ value, onChange }) => (
              <Channel.SelectSingle
                value={value}
                onChange={onChange}
                initialQuery={INDEX_QUERY}
                disabled={isIndex || isVirtual}
                allowNone={false}
                zIndex={100}
              />
            )}
          </Form.Field>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText />
        <Nav.Bar.End align="center" gap="large">
          <Flex.Box x align="center" gap="small">
            <Input.Switch value={createMore} onChange={setCreateMore} />
            <Text.Text color={9}>Create More</Text.Text>
          </Flex.Box>
          <Button.Button
            status={variant}
            variant="filled"
            onClick={() => save()}
            trigger={Triggers.SAVE}
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
