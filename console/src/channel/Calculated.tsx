// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/channel/Calculated.css";

import { channel, TimeSpan } from "@synnaxlabs/client";
import {
  Button,
  Channel,
  Flex,
  Form,
  Input,
  Nav,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { type ReactElement, useRef, useState } from "react";

import { type CalculatedLayoutArgs } from "@/channel/calculatedLayout";
import { Code } from "@/code";
import { Arc } from "@/code/arc";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

const NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  autoFocus: true,
  level: "h2",
  variant: "text",
  placeholder: "Name",
};

export const Calculated: Layout.Renderer = ({ layoutKey, onClose }): ReactElement => {
  const args = Layout.useSelectArgs<CalculatedLayoutArgs>(layoutKey);
  const isEdit = args?.channelKey !== 0;
  const {
    form,
    variant,
    save,
    status: stat,
  } = Channel.useCalculatedForm({
    query: { key: args?.channelKey },
    afterSave: ({ reset }) => {
      if (createMore) reset();
      else onClose();
    },
  });
  const requiresValue = Form.useFieldValue<
    channel.Key[],
    channel.Key[],
    typeof Channel.calculatedFormSchema
  >("requires", { ctx: form, optional: true });
  const isLegacyCalculated = requiresValue != null && requiresValue.length > 0;
  const [createMore, setCreateMore] = useState(false);
  const initialLoaded = useRef(false);
  if (!initialLoaded.current && variant !== "loading") initialLoaded.current = true;
  return (
    <Flex.Box className={CSS.B("channel", "edit", "calculated")} grow empty>
      <Flex.Box className={CSS.B("form")} style={{ padding: "3rem" }} grow>
        <Form.Form<typeof Channel.calculatedFormSchema> {...form}>
          <Form.TextField path="name" label="Name" inputProps={NAME_INPUT_PROPS} />
          {initialLoaded.current && (
            <Form.Field<string> path="expression" grow>
              {({ value, onChange }) => (
                <Code.Editor
                  value={value}
                  language={Arc.LANGUAGE}
                  onChange={onChange}
                  isBlock
                  bordered
                  rounded
                />
              )}
            </Form.Field>
          )}
          {isLegacyCalculated && (
            <Text.Text level="p" status="warning">
              Legacy Calculated Channels are deprecated and will be removed in a future
              release. Please edit this expression to match the new arc-based calculated
              channel syntax.
            </Text.Text>
          )}
          <Flex.Box x>
            <Form.Field<channel.OperationType>
              path="operations.0.type"
              label="Operation"
            >
              {(p) => (
                <Select.Buttons keys={channel.OPERATION_TYPES} {...p}>
                  <Select.Button itemKey="none">None</Select.Button>
                  <Select.Button itemKey="min">Min</Select.Button>
                  <Select.Button itemKey="max">Max</Select.Button>
                  <Select.Button itemKey="avg">Average</Select.Button>
                </Select.Buttons>
              )}
            </Form.Field>
            <Form.Field<TimeSpan>
              path="operations.0.duration"
              label="Window"
              helpText="The value will be reset after this duration. If zero, the value will never be reset."
              grow
            >
              {({ value, onChange }) => (
                <Input.Numeric
                  value={new TimeSpan(value).seconds}
                  onChange={(v) => onChange(TimeSpan.seconds(v))}
                  endContent="S"
                />
              )}
            </Form.Field>
            <Form.Field<channel.Key>
              path="operations.0.resetChannel"
              label="Reset Channel"
              helpText="When this channel is triggered, the calculation will be reset."
            >
              {({ value, onChange }) => (
                <Channel.SelectSingle
                  value={value}
                  onChange={(v: channel.Key | undefined) => onChange(v ?? 0)}
                  grow
                  allowNone
                />
              )}
            </Form.Field>
          </Flex.Box>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Nav.Bar.Start>
          {variant == "success" ? (
            <Triggers.SaveHelpText action={isEdit ? "Save" : "Create"} />
          ) : (
            <Status.Summary status={stat} center={false} />
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End align="center" gap="large">
          {isEdit && (
            <Flex.Box x align="center" gap="small">
              <Input.Switch value={createMore} onChange={setCreateMore} />
              <Text.Text color={9}>Create More</Text.Text>
            </Flex.Box>
          )}
          <Flex.Box x align="center">
            <Button.Button
              status={status.keepVariants(variant, "loading")}
              trigger={Triggers.SAVE}
              variant="filled"
              onClick={() => save()}
            >
              {isEdit ? "Save" : "Create"}
            </Button.Button>
          </Flex.Box>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
