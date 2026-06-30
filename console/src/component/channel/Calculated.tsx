// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/channel/Calculated.css";

import { channel, status, TimeSpan } from "@synnaxlabs/client";
import {
  Arc,
  Button,
  Channel,
  Code,
  Flex,
  Form,
  Icon,
  Input,
  Nav,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useRef, useState } from "react";

import { CSS } from "@/component/css";
import { Modals } from "@/component/modals";
import { Triggers } from "@/component/triggers";

export interface CalculatedModalParams {
  channelKey?: channel.Key;
}

const NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  autoFocus: true,
  level: "h2",
  variant: "text",
  placeholder: "Name",
};

export const useCalculatedModal = Modals.create<CalculatedModalParams>(
  ({ channelKey, close }) => {
    const isEdit = primitive.isNonZero(channelKey);
    const {
      form,
      variant,
      save,
      status: stat,
    } = Channel.useCalculatedForm({
      query: { key: channelKey },
      afterSave: ({ reset }) => {
        if (createMore) reset();
        else close();
      },
    });
    const [createMore, setCreateMore] = useState(false);
    const operationType = Form.useFieldValue<
      channel.OperationType,
      channel.OperationType,
      typeof Channel.calculatedFormSchema
    >("operations.0.type", {
      optional: true,
      ctx: form,
    });
    const initialLoaded = useRef(false);
    if (!initialLoaded.current && variant !== "loading") initialLoaded.current = true;
    const name = Form.useFieldValue<
      string,
      string,
      typeof Channel.calculatedFormSchema
    >("name", { ctx: form });
    return (
      <Modals.Frame className={CSS.B("channel", "edit", "calculated")}>
        <Modals.Header icon={<Icon.Channel />}>
          {isEdit ? `${name}.Edit` : "Channel.Create.Calculated"}
        </Modals.Header>
        <Modals.Body justify="start" gap="large">
          <Form.Form<typeof Channel.calculatedFormSchema> {...form}>
            <Form.TextField path="name" label="Name" inputProps={NAME_INPUT_PROPS} />
            {initialLoaded.current && (
              <Form.Field<string> path="expression" grow>
                {({ value, onChange }) => (
                  <Flex.Box grow className={CSS.B("calculated", "editor")}>
                    <Code.Editor
                      initialValue={value}
                      language={Arc.NAME}
                      onValueChange={onChange}
                      isBlock
                      bordered
                      rounded
                    />
                  </Flex.Box>
                )}
              </Form.Field>
            )}
            <Flex.Box x className={CSS.B("operations")} align="start">
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
                    <Select.Button itemKey="derivative">Derivative</Select.Button>
                  </Select.Buttons>
                )}
              </Form.Field>
              {operationType !== "derivative" && (
                <>
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
                        endContent="s"
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
                </>
              )}
            </Flex.Box>
          </Form.Form>
        </Modals.Body>
        <Modals.Footer>
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
        </Modals.Footer>
      </Modals.Frame>
    );
  },
);
