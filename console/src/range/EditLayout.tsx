// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useRef } from "react";

import { TimeRange, TimeStamp, UnexpectedError } from "@synnaxlabs/client";
import { Icon, Logo } from "@synnaxlabs/media";
import { Align, Button, Form, Nav, Synnax, Text } from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { useSelect } from "@/range/selectors";
import { add } from "@/range/slice";

import "@/range/EditLayout.css";

const formSchema = z.object({
  name: z.string().min(1, "Name must not be empty"),
  start: z.number().int(),
  end: z.number().int(),
  labels: z.string().array(),
});

const CREATE_RANGE_WINDOW_KEY = "defineRange";

export const editLayout = (name: string = "Create Range"): Layout.LayoutState => ({
  key: CREATE_RANGE_WINDOW_KEY,
  type: CREATE_RANGE_WINDOW_KEY,
  windowKey: CREATE_RANGE_WINDOW_KEY,
  name,
  location: "window",
  window: {
    resizable: false,
    size: { height: 280, width: 700 },
    navTop: true,
    transparent: true,
  },
});

type DefineRangeFormProps = z.infer<typeof formSchema>;

export const EditLayout = (props: Layout.RendererProps): ReactElement => {
  const { layoutKey } = props;
  const now = useRef(Number(TimeStamp.now().valueOf())).current;
  const range = useSelect(layoutKey);
  const client = Synnax.use();
  const isCreate = layoutKey === CREATE_RANGE_WINDOW_KEY;
  const isRemoteEdit = !isCreate && (range == null || range.persisted);
  const initialValues = useQuery<DefineRangeFormProps>({
    queryKey: ["range", layoutKey],
    queryFn: async () => {
      if (isCreate)
        return {
          name: "",
          start: now,
          end: now,
          labels: [],
        };
      if (range == null || range.persisted) {
        if (client == null) throw new UnexpectedError("Client is not available");
        const rng = await client.ranges.retrieve(layoutKey);
        return {
          name: rng.name,
          start: Number(rng.timeRange.start.valueOf()),
          end: Number(rng.timeRange.end.valueOf()),
          labels: [],
        };
      }
      if (range.variant !== "static") throw new UnexpectedError("Range is not static");
      return {
        name: range.name,
        start: range.timeRange.start,
        end: range.timeRange.end,
        labels: [],
      };
    },
  });
  if (initialValues.isPending) return <Logo.Watermark variant="loader" />;
  if (initialValues.isError) throw initialValues.error;
  return (
    <EditLayoutForm
      isRemoteEdit={isRemoteEdit}
      initialValues={initialValues.data}
      {...props}
    />
  );
};

interface EditLayoutFormProps extends Layout.RendererProps {
  initialValues: DefineRangeFormProps;
  isRemoteEdit: boolean;
  onClose: () => void;
}

const EditLayoutForm = ({
  layoutKey,
  initialValues,
  isRemoteEdit,
  onClose,
}: EditLayoutFormProps): ReactElement => {
  const methods = Form.use({ values: initialValues, schema: formSchema });
  const dispatch = useDispatch();
  const client = Synnax.use();
  const isCreate = layoutKey === CREATE_RANGE_WINDOW_KEY;

  const { mutate, isPending } = useMutation({
    mutationFn: async (persist: boolean) => {
      if (!methods.validate()) return;
      let { start, end, name } = methods.value();
      const startTS = new TimeStamp(start, "UTC");
      const endTS = new TimeStamp(end, "UTC");
      name = name.trim();
      const key = isCreate ? uuidv4() : layoutKey;
      const persisted = persist || isRemoteEdit;
      const tr = new TimeRange(startTS, endTS);
      if (persisted && client != null)
        await client.ranges.create({ key, name, timeRange: tr });
      dispatch(
        add({
          ranges: [
            {
              variant: "static",
              name,
              timeRange: {
                start: Number(startTS.valueOf()),
                end: Number(endTS.valueOf()),
              },
              key,
              persisted,
            },
          ],
        }),
      );
      onClose();
    },
  });

  return (
    <Align.Space className={CSS.B("range-edit-layout")} grow>
      <Align.Space className="console-form" justify="center" grow>
        <Form.Form {...methods}>
          <Form.Field<string> path="name">
            {(p) => (
              <Input.Text
                autoFocus
                level="h2"
                variant="natural"
                placeholder="Range Name"
                {...p}
              />
            )}
          </Form.Field>
          <Align.Space direction="x" size="large">
            <Form.Field<number> path="start" label="From">
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Form.Field>
            <Text.WithIcon level="h4" startIcon={<Icon.Arrow.Right />} />
            <Form.Field<number> path="end" label="To">
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Form.Field>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          {(isCreate || !isRemoteEdit) && (
            <Button.Button
              onClick={() => mutate(true)}
              variant="outlined"
              disabled={client == null || isPending}
              loading={isPending}
            >
              Save to Synnax
            </Button.Button>
          )}
          <Button.Button onClick={() => mutate(false)}>
            Save {!isRemoteEdit && "Locally"}
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
