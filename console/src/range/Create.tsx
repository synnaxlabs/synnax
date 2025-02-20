// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Create.css";

import { ranger, TimeRange, TimeStamp } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Form,
  Icon as PIcon,
  Input,
  Nav,
  Ranger,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { deep, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useRef } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { Label } from "@/label";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { add } from "@/range/slice";
import { Triggers } from "@/triggers";

const formSchema = z.object({
  key: z.string().optional(),
  name: z.string().min(1, "Name must not be empty"),
  timeRange: z.object({ start: z.number(), end: z.number() }),
  labels: z.string().array(),
  parent: z.string().optional(),
});

export type FormProps = z.infer<typeof formSchema>;

export type CreateLayoutArgs = Partial<FormProps>;

export const CREATE_LAYOUT_TYPE = "editRange";

export const CREATE_LAYOUT: Layout.BaseState<CreateLayoutArgs> = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  location: "modal",
  name: "Range.Create",
  icon: "Range",
  window: {
    resizable: false,
    size: { height: 370, width: 700 },
    navTop: true,
  },
};

export const createCreateLayout = (
  initial: CreateLayoutArgs,
): Layout.BaseState<CreateLayoutArgs> => ({
  ...CREATE_LAYOUT,
  args: initial,
});

const parentRangeIcon = (
  <PIcon.Icon bottomRight={<Icon.Arrow.Up />}>
    <Icon.Range />
  </PIcon.Icon>
);

export const Create = (props: Layout.RendererProps) => {
  const { layoutKey } = props;
  const now = useRef(Number(TimeStamp.now().valueOf())).current;
  const args = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);
  const initialValues: FormProps = {
    name: "",
    labels: [],
    timeRange: { start: now, end: now },
    parent: "",
    ...args,
  };

  return <CreateLayoutForm initialValues={initialValues} {...props} />;
};

interface CreateLayoutFormProps extends Layout.RendererProps {
  initialValues: FormProps;
  onClose: () => void;
}

const CreateLayoutForm = ({ initialValues, onClose }: CreateLayoutFormProps) => {
  const methods = Form.use({ values: deep.copy(initialValues), schema: formSchema });
  const dispatch = useDispatch();
  const client = Synnax.use();
  const clientExists = client != null;
  const handleException = Status.useExceptionHandler();

  const { mutate, isPending } = useMutation({
    mutationFn: async (persisted: boolean) => {
      if (!methods.validate()) return;
      const values = methods.value();
      const { timeRange: tr, parent } = values;
      const timeRange = new TimeRange(tr);
      const name = values.name.trim();
      const key = initialValues.key ?? uuidv4();
      const parentID = primitiveIsZero(parent)
        ? undefined
        : ranger.ontologyID(parent as string);
      const otgID = ranger.ontologyID(key);
      if (persisted && clientExists) {
        await client.ranges.create({ key, name, timeRange }, { parent: parentID });
        await client.labels.label(otgID, values.labels, { replace: true });
      }
      dispatch(
        add({
          ranges: [
            { variant: "static", name, timeRange: timeRange.numeric, key, persisted },
          ],
        }),
      );
      onClose();
    },
    onError: (e) => handleException(e, "Failed to create range"),
  });

  // Makes sure the user doesn't have the option to select the range itself as a parent
  const recursiveParentFilter = useCallback(
    (data: ranger.Payload[]) => data.filter((r) => r.key !== initialValues.key),
    [initialValues.key],
  );

  return (
    <Align.Space className={CSS.B("range-create-layout")} grow empty>
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
          <Align.Space direction="x">
            <Form.Field<string> path="parent" visible padHelpText={false}>
              {({ onChange, ...p }) => (
                <Ranger.SelectSingle
                  dropdownVariant="modal"
                  style={{ width: "fit-content" }}
                  zIndex={100}
                  filter={recursiveParentFilter}
                  entryRenderKey={(e) => (
                    <Text.WithIcon
                      level="p"
                      shade={9}
                      startIcon={parentRangeIcon}
                      size="small"
                    >
                      {e.name}
                    </Text.WithIcon>
                  )}
                  inputPlaceholder="Search Ranges"
                  triggerTooltip="Select Parent Range"
                  placeholder={
                    <Text.WithIcon
                      level="p"
                      shade={7}
                      startIcon={parentRangeIcon}
                      size="small"
                    >
                      Parent Range
                    </Text.WithIcon>
                  }
                  onChange={(v: string) => onChange(v ?? "")}
                  {...p}
                />
              )}
            </Form.Field>
            <Form.Field<string[]> path="labels" required={false}>
              {({ variant, ...p }) => (
                <Label.SelectMultiple
                  entryRenderKey="name"
                  dropdownVariant="floating"
                  zIndex={100}
                  location="bottom"
                  {...p}
                />
              )}
            </Form.Field>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action="Save to Synnax" />
        <Nav.Bar.End>
          <Button.Button
            variant="outlined"
            onClick={() => mutate(false)}
            disabled={isPending}
          >
            Save Locally
          </Button.Button>
          <Button.Button
            onClick={() => mutate(true)}
            disabled={!clientExists || isPending}
            tooltip={clientExists ? "Save to Cluster" : "No Cluster Connected"}
            tooltipLocation="bottom"
            loading={isPending}
            triggers={Triggers.SAVE}
          >
            Save to Synnax
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Align.Space>
  );
};
