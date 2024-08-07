// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/EditLayout.css";

import { TimeRange, TimeStamp, UnexpectedError } from "@synnaxlabs/client";
import { Icon, Logo } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Form,
  Nav,
  Ranger,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useRef } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { type TimeRange as TimeRangeT } from "@/range/migrations";
import { useSelect } from "@/range/selectors";
import { add } from "@/range/slice";

const formSchema = z.object({
  name: z.string().min(1, "Name must not be empty"),
  timeRange: z.object({
    start: z.number().int(),
    end: z.number().int(),
  }),
  labels: z.string().array(),
  parent: z.string().optional(),
});

export const EDIT_LAYOUT_TYPE = "editRange";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const createEditLayout = (
  name: string = "Range.Create",
  timeRange?: TimeRangeT,
): Layout.State => ({
  key: EDIT_LAYOUT_TYPE,
  type: EDIT_LAYOUT_TYPE,
  windowKey: EDIT_LAYOUT_TYPE,
  name,
  icon: "Range",
  location: "modal",
  window: {
    resizable: false,
    size: { height: 400, width: 700 },
    navTop: true,
  },
  args: timeRange,
});

type DefineRangeFormProps = z.infer<typeof formSchema>;

export const Edit = (props: Layout.RendererProps): ReactElement => {
  const { layoutKey } = props;
  const now = useRef(Number(TimeStamp.now().valueOf())).current;
  const range = useSelect(layoutKey);
  const args = Layout.useSelectArgs<TimeRangeT>(layoutKey);
  const client = Synnax.use();
  const isCreate = layoutKey === EDIT_LAYOUT_TYPE;
  const isRemoteEdit = !isCreate && (range == null || range.persisted);
  const initialValues = useQuery<DefineRangeFormProps>({
    queryKey: ["range", layoutKey],
    queryFn: async () => {
      if (isCreate) {
        return {
          name: "",
          labels: [],
          timeRange: args ?? {
            start: now,
            end: now,
          },
          parent: "",
        };
      }
      if (range == null || range.persisted) {
        if (client == null) throw new UnexpectedError("Client is not available");
        const rng = await client.ranges.retrieve(layoutKey);
        return {
          name: rng.name,
          timeRange: {
            start: Number(rng.timeRange.start.valueOf()),
            end: Number(rng.timeRange.end.valueOf()),
          },
          labels: [],
        };
      }
      if (range.variant !== "static") throw new UnexpectedError("Range is not static");
      return {
        name: range.name,
        timeRange: range.timeRange,
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
  const methods = Form.use({ values: initialValues, schema: formSchema, sync: true });
  const dispatch = useDispatch();
  const client = Synnax.use();
  const isCreate = layoutKey === EDIT_LAYOUT_TYPE;

  const { mutate, isPending } = useMutation({
    mutationFn: async (persist: boolean) => {
      if (!methods.validate()) return;
      const values = methods.value();
      const { timeRange } = methods.value();
      const startTS = new TimeStamp(timeRange.start, "UTC");
      const endTS = new TimeStamp(timeRange.end, "UTC");
      const name = values.name.trim();
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
      <Align.Space
        className="console-form"
        justify="center"
        style={{ padding: "1rem 3rem" }}
        grow
      >
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
            <Form.Field<number> path="timeRange.start" label="From">
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Form.Field>
            <Text.WithIcon level="h4" startIcon={<Icon.Arrow.Right />} />
            <Form.Field<number> path="timeRange.end" label="To">
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Form.Field>
          </Align.Space>
          <Form.Field<string> showLabel={false} path="parent">
            {(p) => (
              <Ranger.SelectSingle
                dropdownVariant="modal"
                style={{ width: "fit-content" }}
                entryRenderKey={(e) => (
                  <Text.WithIcon
                    level="p"
                    shade={9}
                    startIcon={<Icon.Arrow.Up />}
                    size="small"
                  >
                    {e.name}
                  </Text.WithIcon>
                )}
                placeholder={
                  <Text.WithIcon
                    level="p"
                    shade={7}
                    startIcon={<Icon.Arrow.Up />}
                    size="small"
                  >
                    Parent Range
                  </Text.WithIcon>
                }
                allowNone={false}
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
        <Nav.Bar.End style={{ padding: "1rem" }}>
          <Button.Button variant="outlined" onClick={() => mutate(false)}>
            Save {!isRemoteEdit && "Locally"}
          </Button.Button>
          {(isCreate || !isRemoteEdit) && (
            <Button.Button
              onClick={() => mutate(true)}
              disabled={client == null || isPending}
              loading={isPending}
              triggers={[SAVE_TRIGGER]}
            >
              Save to Synnax
            </Button.Button>
          )}
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
