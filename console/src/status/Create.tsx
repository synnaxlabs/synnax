// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form, Status } from "@synnaxlabs/pluto";
import { type status, TimeStamp, uuid } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import type z from "zod";

import { Layout } from "@/layout";

export type CreateLayoutArgs = Partial<z.infer<typeof Status.formSchema>>;

export const CREATE_LAYOUT_TYPE = "createStatus";

export const CREATE_LAYOUT: Layout.BaseState<CreateLayoutArgs> = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  location: "modal",
  name: "Status.Create",
  icon: "Status",
  window: { resizable: false, size: { height: 440, width: 700 }, navTop: true },
};

export const createCreateLayout = (
  initial: CreateLayoutArgs = {},
): Layout.BaseState<CreateLayoutArgs> => ({ ...CREATE_LAYOUT, args: initial });

export const Create = ({ layoutKey, onClose }: Layout.RendererProps): ReactElement => {
  const args = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);
  const { form } = Status.useForm({
    params: { key: args?.key },
    autoSave: false,
    initialValues: {
      ...args,
      key: uuid.create(),
      message: "",
      time: TimeStamp.now(),
      name: "",
      description: "",
      details: undefined,
      variant: "success",
    },
    afterSave: onClose,
  });

  return (
    <Flex.Box grow empty>
      <Form.Form<typeof Status.formSchema> {...form}>
        <Form.TextField
          path="name"
          inputProps={{
            autoFocus: true,
            level: "h2",
            variant: "text",
            placeholder: "Name",
          }}
        />
      </Form.Form>
      <Form.Field<status.Variant> path="variant" label="Variant">
        {({ value, onChange }) => (
          <Status.SelectVariant value={value} onChange={onChange} />
        )}
      </Form.Field>
    </Flex.Box>
  );
};
