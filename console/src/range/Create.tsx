// Copyright 2025 Synnax Labs, Inc.
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
  Align,
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
import { useCallback, useRef } from "react";
import { type z } from "zod";

import { CSS } from "@/css";
import { Label } from "@/label";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export type CreateLayoutArgs = Partial<z.infer<typeof Ranger.rangeFormSchema>>;

export const CREATE_LAYOUT_TYPE = "editRange";

export const CREATE_LAYOUT: Layout.BaseState<CreateLayoutArgs> = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  location: "modal",
  name: "Range.Create",
  icon: "Range",
  window: {
    resizable: false,
    size: { height: 440, width: 700 },
    navTop: true,
  },
};

export const createCreateLayout = (
  initial: CreateLayoutArgs = {},
): Layout.BaseState<CreateLayoutArgs> => ({
  ...CREATE_LAYOUT,
  args: initial,
});

export const ParentRangeIcon = Icon.createComposite(Icon.Range, {
  bottomRight: Icon.Arrow.Up,
});

export const Create: Layout.Renderer = (props) => {
  const { layoutKey, onClose } = props;
  const now = useRef(Number(TimeStamp.now().valueOf())).current;
  const args = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);

  const client = Synnax.use();
  const clientExists = client != null;
  const { form, save, variant } = Ranger.useForm({
    params: { key: args?.key },
    autoSave: false,
    initialValues: {
      key: uuid.create(),
      name: "",
      labels: [],
      stage: "to_do",
      timeRange: { start: now, end: now },
      parent: "",
      ...args,
    },
    afterSave: () => onClose(),
  });

  // Makes sure the user doesn't have the option to select the range itself as a parent
  const recursiveParentFilter = useCallback(
    (data: ranger.Payload) => data.key !== args?.key,
    [args?.key],
  );

  return (
    <Align.Space className={CSS.B("range-create-layout")} grow empty>
      <Align.Space
        className="console-form"
        justify="center"
        style={{ padding: "1rem 3rem" }}
        grow
      >
        <Form.Form<typeof Ranger.rangeFormSchema> {...form}>
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
          <Form.Field<ranger.Stage> path="stage" required={false}>
            {(p) => (
              <Ranger.SelectStage {...p} variant="outlined" style={{ width: 150 }} />
            )}
          </Form.Field>
          <Align.Space x size="large">
            <Form.Field<number> path="timeRange.start" label="From" required={false}>
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Form.Field>
            <Text.WithIcon level="h4" startIcon={<Icon.Arrow.Right />} />
            <Form.Field<number> path="timeRange.end" label="To" required={false}>
              {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space x>
            <Form.Field<string> path="parent" visible padHelpText={false}>
              {({ onChange, value }) => (
                <Ranger.SelectSingle
                  style={{ width: "fit-content" }}
                  zIndex={100}
                  filter={recursiveParentFilter}
                  value={value}
                  onChange={(v: ranger.Key) => onChange(v ?? "")}
                  icon={<ParentRangeIcon />}
                  allowNone
                />
              )}
            </Form.Field>
            <Form.Field<string[]> path="labels" required={false}>
              {({ variant, ...p }) => (
                <Label.SelectMultiple zIndex={100} location="bottom" {...p} />
              )}
            </Form.Field>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action="Save to Synnax" />
        <Nav.Bar.End>
          <Button.Button onClick={() => save()} disabled={variant === "loading"}>
            Save Locally
          </Button.Button>
          <Button.Button
            variant="filled"
            onClick={() => save()}
            disabled={!clientExists || variant === "loading"}
            tooltip={clientExists ? "Save to Cluster" : "No Cluster Connected"}
            tooltipLocation="bottom"
            loading={variant === "loading"}
            triggers={Triggers.SAVE}
          >
            Save to Synnax
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Align.Space>
  );
};
