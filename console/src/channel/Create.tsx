// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { DataType, Rate, channel } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Channel,
  Form,
  Input,
  Nav,
  Select,
  Synnax,
} from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useRef, useEffect } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

export const CREATE_LAYOUT_TYPE = "createChannel";

export const createLayout: Layout.State = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  windowKey: CREATE_LAYOUT_TYPE,
  name: "Create Channel",
  location: "window",
  window: {
    resizable: false,
    size: { height: 550, width: 700 },
    navTop: true,
    transparent: true,
  },
};

export const Create: Layout.Renderer = ({
  layoutKey,
  onClose,
}: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    schema: channel.payload
      .extend({
        name: z.string().min(1, "Name must not be empty"),
        dataType: DataType.z.transform((v) => v.toString()),
      })
      .refine(
        (v) => !v.isIndex || new DataType(v.dataType).equals(DataType.TIMESTAMP),
        {
          message: "Index channel must have data type TIMESTAMP",
          path: ["dataType"],
        },
      )
      .refine((v) => v.isIndex || v.index !== 0, {
        message: "Data channel must have an index",
        path: ["index"],
      }),
    values: {
      key: 0,
      name: "",
      index: 0,
      dataType: "float32",
      isIndex: false,
      leaseholder: 0,
      rate: Rate.hz(0),
    },
  });

  const { mutate, isPending, isError, error } = useMutation({
    mutationFn: async () => {
      console.log(methods.validate());
      if (!methods.validate() || client == null) return;
      const d = methods.value();
      d.dataType = d.dataType.toString();
      await client.channels.create(methods.value());
    },
  });

  const isIndex = Form.useFieldValue("isIndex", false, methods);

  return (
    <Align.Space className={CSS.B("channel-edit-layout")} grow>
      <Align.Space className="console-form" style={{ padding: "5rem 3rem" }} grow>
        <Form.Form {...methods}>
          <Form.Field<string> path="name" label="Name">
            {(p) => (
              <Input.Text
                autoFocus
                level="h2"
                variant="natural"
                placeholder="Channel Name"
                {...p}
              />
            )}
          </Form.Field>
          <Align.Space direction="x" size="large">
            <Form.SwitchField
              path="isIndex"
              label="Is Index"
              onChange={(v, ctx) => {
                if (v)
                  ctx.set({ path: "dataType", value: DataType.TIMESTAMP.toString() });
              }}
            />
            <Form.Field<DataType> path="dataType" label="Data Type" grow>
              {(p) => <Select.DataType {...p} disabled={isIndex} />}
            </Form.Field>
          </Align.Space>
          <Form.Field<channel.Key> path="index" label="Index">
            {(p) => (
              <Channel.SelectSingle
                client={client}
                placeholder="Select Index"
                searchOptions={{ isIndex: true }}
                disabled={isIndex}
                {...p}
              />
            )}
          </Form.Field>
        </Form.Form>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          <Button.Button
            disabled={isPending}
            loading={isPending}
            onClick={() => {
              mutate();
            }}
          >
            Create Channel
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
