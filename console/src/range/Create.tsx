// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Create.css";

import { type ranger, TimeStamp } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Form,
  Icon,
  Input,
  Nav,
  Ranger,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type NumericTimeRange, TimeRange, uuid } from "@synnaxlabs/x";
import { useCallback, useRef } from "react";
import { useDispatch } from "react-redux";
import { type z } from "zod";

import { CSS } from "@/css";
import { Label } from "@/label";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { add } from "@/range/slice";
import { Triggers } from "@/triggers";

export type CreateLayoutArgs = Partial<z.infer<typeof Ranger.formSchema>>;

export const CREATE_LAYOUT_TYPE = "editRange";

export const CREATE_LAYOUT: Layout.BaseState<CreateLayoutArgs> = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  location: "modal",
  name: "Range.Create",
  icon: "Range",
  window: { resizable: false, size: { height: 440, width: 700 }, navTop: true },
};

export const createCreateLayout = (
  initial: CreateLayoutArgs = {},
): Layout.BaseState<CreateLayoutArgs> => ({ ...CREATE_LAYOUT, args: initial });

export const ParentRangeIcon = Icon.createComposite(Icon.Range, {
  bottomRight: Icon.Arrow.Up,
});

export const Create: Layout.Renderer = (props) => {
  const { layoutKey, onClose } = props;
  const now = useRef(Number(TimeStamp.now().valueOf())).current;
  const args = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);
  const dispatch = useDispatch();

  const client = Synnax.use();
  const clientExists = client != null;
  const { form, save, variant } = Ranger.useForm({
    query: { key: args?.key },
    autoSave: false,
    initialValues: {
      key: uuid.create(),
      name: "",
      labels: [],
      timeRange: { start: now, end: now },
      parent: "",
      ...args,
    },
    afterSave: (form) => {
      onClose();
      const { name, key, timeRange } = form.value();
      if (key == null) return;
      dispatch(
        add({
          ranges: [{ name, key, persisted: true, variant: "static", timeRange }],
        }),
      );
    },
  });

  const saveLocal = useCallback(() => {
    if (!form.validate()) return;
    const value = form.value();
    if (value.key == null) return;
    dispatch(
      add({
        ranges: [
          {
            persisted: false,
            ...value,
            key: value.key ?? "",
            variant: "static",
            timeRange: new TimeRange(value.timeRange.start, value.timeRange.end)
              .numeric,
          },
        ],
      }),
    );
    onClose();
  }, [form, dispatch]);

  // Makes sure the user doesn't have the option to select the range itself as a parent
  const recursiveParentFilter = useCallback(
    (data: ranger.Payload) => data.key !== args?.key,
    [args?.key],
  );

  const saveName = "Save to Synnax";

  return (
    <Flex.Box className={CSS.B("range-create-layout")} grow empty>
      <Flex.Box
        className="console-form"
        justify="center"
        style={{ padding: "1rem 3rem" }}
        grow
      >
        <Form.Form<typeof Ranger.formSchema> {...form}>
          <Form.Field<string> path="name">
            {(p) => (
              <Input.Text
                autoFocus
                level="h2"
                variant="text"
                placeholder="Range Name"
                {...p}
              />
            )}
          </Form.Field>
          <Form.Field<NumericTimeRange> path="timeRange" label="Stage">
            {(p) => (
              <Ranger.SelectStage
                {...Ranger.wrapNumericTimeRangeToStage(p)}
                style={{ width: 150 }}
                triggerProps={{ variant: "outlined" }}
              />
            )}
          </Form.Field>
          <Flex.Box x gap="large">
            <Form.Field<number> path="timeRange.start" label="From">
              {(p) => <Input.DateTime level="h4" variant="text" {...p} />}
            </Form.Field>
            <Text.Text level="h4">
              <Icon.Arrow.Right />
            </Text.Text>
            <Form.Field<number> path="timeRange.end" label="To">
              {(p) => <Input.DateTime level="h4" variant="text" {...p} />}
            </Form.Field>
          </Flex.Box>
          <Flex.Box x>
            <Form.Field<string> path="parent" visible padHelpText={false}>
              {({ onChange, value }) => (
                <Ranger.Select
                  style={{ width: "fit-content" }}
                  zIndex={-1}
                  filter={recursiveParentFilter}
                  value={value}
                  onChange={onChange}
                  icon={<ParentRangeIcon />}
                  allowNone
                />
              )}
            </Form.Field>
            <Form.Field<string[]> path="labels" required={false}>
              {({ variant, ...p }) => <Label.SelectMultiple zIndex={100} {...p} />}
            </Form.Field>
          </Flex.Box>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action={saveName} />
        <Nav.Bar.End>
          <Button.Button onClick={() => saveLocal()} disabled={variant === "loading"}>
            Save Locally
          </Button.Button>
          <Button.Button
            variant="filled"
            onClick={() => save()}
            disabled={!clientExists}
            status={variant}
            trigger={Triggers.SAVE}
          >
            {saveName}
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
