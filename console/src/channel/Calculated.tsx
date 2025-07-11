// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Channel,
  Form,
  Input,
  Nav,
  Select,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { unique } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { type CalculatedLayoutArgs } from "@/channel/calculatedLayout";
import { Code } from "@/code";
import { Lua } from "@/code/lua";
import {
  usePhantomGlobals,
  type UsePhantomGlobalsReturn,
  type Variable,
} from "@/code/phantom";
import { bindChannelsAsGlobals, useSuggestChannels } from "@/code/useSuggestChannels";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

const FAILED_TO_UPDATE_AUTOCOMPLETE =
  "Failed to update calculated channel auto-complete";

const GLOBALS: Variable[] = [
  {
    key: "get",
    name: "get",
    value: `
    -- Get a channel's value by its name. This function should be used when
    -- the channel name cannot be used directly as a variable. For example,
    -- hyphenated names such as 'my-channel' should be accessed with get("my-channel")
    -- instead of just my-channel.
    function get(name)
    end
    `,
  },
];

export const Calculated: Layout.Renderer = ({ layoutKey }): ReactElement => {
  const client = Synnax.use();
  const { channelKey } = Layout.useSelectArgs<CalculatedLayoutArgs>(layoutKey);
  const isEdit = channelKey !== 0;

  const { form, variant, save } = Channel.useCalculatedForm({
    params: { key: channelKey },
  });

  const handleError = Status.useErrorHandler();
  const [createMore, setCreateMore] = useState(false);

  const isIndex = Form.useFieldValue<boolean, boolean, typeof Channel.calculatedFormSchema>(
    "isIndex", { ctx: form }
  );

  const globals = usePhantomGlobals({
    language: Lua.LANGUAGE,
    stringifyVar: Lua.stringifyVar,
    initialVars: GLOBALS,
  });
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const channels = form.get<channel.Key[]>("requires").value;
      try {
        const chs = await client.channels.retrieve(channels);
        if (signal.aborted) return;
        chs.forEach((ch) => globals.set(ch.key.toString(), ch.name, ch.key.toString()));
      } catch (e) {
        handleError(e, FAILED_TO_UPDATE_AUTOCOMPLETE);
      }
    },
    [form, globals, client],
  );

  return (
    <Align.Space className={CSS.B("channel-edit-layout")} grow empty>
      <Align.Space className="console-form" style={{ padding: "3rem" }} grow>
        <Form.Form<typeof Channel.calculatedFormSchema> {...form}>
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

          <Form.Field<string> path="expression" grow>
            {({ value, onChange }) => (
              <Editor
                value={value}
                language={Lua.LANGUAGE}
                onChange={onChange}
                bordered
                rounded
                style={{ height: 150 }}
                globals={globals}
              />
            )}
          </Form.Field>
          <Align.Space x>
            <Form.Field<string>
              path="dataType"
              label="Output Data Type"
              style={{ width: 150 }}
            >
              {({ variant: _, ...p }) => (
                <Select.DataType
                  {...p}
                  disabled={isIndex}
                  maxHeight="small"
                  zIndex={100}
                  style={{ width: 150 }}
                />
              )}
            </Form.Field>
            <Form.Field<channel.Key[]>
              path="requires"
              required
              label="Required Channels"
              grow
              onChange={(v, extra) => {
                if (client == null) return;
                handleError(
                  async () =>
                    await bindChannelsAsGlobals(
                      client,
                      extra.get<channel.Key[]>("requires").value,
                      v,
                      globals,
                    ),
                  FAILED_TO_UPDATE_AUTOCOMPLETE,
                );
              }}
            >
              {({ variant: _, ...p }) => <Channel.SelectMultiple zIndex={100} {...p} />}
            </Form.Field>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action={isEdit ? "Save" : "Create"} />
        <Nav.Bar.End align="center" size="large">
          {isEdit && (
            <Align.Space x align="center" size="small">
              <Input.Switch value={createMore} onChange={setCreateMore} />
              <Text.Text level="p" shade={11}>
                Create More
              </Text.Text>
            </Align.Space>
          )}
          <Align.Space x align="center">
            <Button.Button
              disabled={variant === "loading"}
              loading={variant === "loading"}
              triggers={Triggers.SAVE}
              onClick={() => save()}
            >
              {isEdit ? "Save" : "Create"}
            </Button.Button>
          </Align.Space>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Align.Space>
  );
};

interface EditorProps extends Code.EditorProps {
  globals?: UsePhantomGlobalsReturn;
}

const Editor = ({ globals, ...props }: EditorProps): ReactElement => {
  const methods = Form.useContext();
  const onAccept = useCallback(
    (channel: channel.Payload) => {
      if (globals == null) return;
      globals.set(channel.key.toString(), channel.name, channel.key.toString());
      methods.set(
        "requires",
        unique.unique([...methods.get<channel.Key[]>("requires").value, channel.key]),
      );
    },
    [methods, globals],
  );

  useSuggestChannels(onAccept);

  return <Code.Editor {...props} />;
};
