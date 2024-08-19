// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, Rate } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Channel,
  Form,
  Input,
  Nav,
  Select,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Layout } from "@/layout";

export const CREATE_LAYOUT_TYPE = "createChannel";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const createLayout: Layout.State = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  windowKey: CREATE_LAYOUT_TYPE,
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

const schema = channel.newPayload
  .extend({
    name: z.string().min(1, "Name must not be empty"),
    dataType: DataType.z.transform((v) => v.toString()),
  })
  .refine((v) => !v.isIndex || new DataType(v.dataType).equals(DataType.TIMESTAMP), {
    message: "Index channel must have data type TIMESTAMP",
    path: ["dataType"],
  })
  .refine((v) => v.isIndex || v.index !== 0 || v.virtual, {
    message: "Data channel must have an index",
    path: ["index"],
  });

export const CreateModal: Layout.Renderer = ({ onClose }): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use<typeof schema>({
    schema,
    values: {
      key: 0,
      name: "",
      index: 0,
      dataType: "float32",
      internal: false,
      isIndex: false,
      leaseholder: 0,
      rate: Rate.hz(0),
      virtual: false,
    },
  });
  const [createMore, setCreateMore] = useState(false);

  const { mutate, isPending } = useMutation({
    mutationFn: async (createMore: boolean) => {
      console.log(methods.validate(), methods.value());
      if (!methods.validate() || client == null) return;
      const d = methods.value();
      d.dataType = d.dataType.toString();
      await client.channels.create(methods.value());
      if (!createMore) onClose();
      else
        methods.reset({
          key: 0,
          name: "",
          index: 0,
          dataType: "float32",
          isIndex: false,
          leaseholder: 0,
          virtual: false,
          rate: Rate.hz(0),
          internal: false,
        });
    },
  });

  const isIndex = Form.useFieldValue<boolean, boolean, typeof schema>(
    "isIndex",
    false,
    methods,
  );
  const isVirtual = Form.useFieldValue<boolean, boolean, typeof schema>(
    "virtual",
    false,
    methods,
  );

  return (
    <Align.Space className={CSS.B("channel-edit-layout")} grow empty>
      <Align.Space className="console-form" style={{ padding: "3rem" }} grow>
        <Form.Form {...methods}>
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
          <Align.Space direction="x" size="large">
            <Form.SwitchField
              path="virtual"
              label="Virtual"
              inputProps={{ disabled: isIndex }}
              onChange={(v, ctx) => {
                if (!v) return;
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
            <Form.Field<DataType> path="dataType" label="Data Type" grow>
              {(p) => (
                <Select.DataType
                  {...p}
                  disabled={isIndex}
                  maxHeight="small"
                  zIndex={100}
                />
              )}
            </Form.Field>
          </Align.Space>
          <Form.Field<channel.Key> path="index" label="Index">
            {(p) => (
              <Channel.SelectSingle
                client={client}
                placeholder="Select Index"
                searchOptions={{ isIndex: true }}
                disabled={isIndex || isVirtual}
                maxHeight="small"
                allowNone={false}
                zIndex={100}
                {...p}
              />
            )}
          </Form.Field>
        </Form.Form>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }} size="small">
          <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
          <Text.Text shade={7} level="small">
            To Save
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ padding: "1rem" }} align="center" size="large">
          <Align.Space direction="x" align="center" size="small">
            <Input.Switch value={createMore} onChange={setCreateMore} />
            <Text.Text level="p" shade={7}>
              Create More
            </Text.Text>
          </Align.Space>
          <Button.Button
            disabled={isPending}
            loading={isPending}
            onClick={() => mutate(createMore)}
            triggers={[SAVE_TRIGGER]}
          >
            Create Channel
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
