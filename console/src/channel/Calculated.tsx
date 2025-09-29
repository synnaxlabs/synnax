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
  Button,
  Channel,
  Flex,
  Form,
  Input,
  Nav,
  Status,
  Synnax,
  Telem,
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

export const Calculated: Layout.Renderer = ({ layoutKey, onClose }): ReactElement => {
  const client = Synnax.use();
  const args = Layout.useSelectArgs<CalculatedLayoutArgs>(layoutKey);
  const isEdit = args?.channelKey !== 0;

  const { form, variant, save, status } = Channel.useCalculatedForm({
    query: { key: args?.channelKey },
    afterSave: ({ reset }) => {
      if (createMore) reset();
      else onClose();
    },
  });

  const handleError = Status.useErrorHandler();
  const [createMore, setCreateMore] = useState(false);

  const isIndex = Form.useFieldValue<
    boolean,
    boolean,
    typeof Channel.calculatedFormSchema
  >("isIndex", { ctx: form });

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

  if (variant !== "success") return <Status.Summary status={status} />;

  return (
    <Flex.Box className={CSS.B("channel-edit-layout")} grow empty>
      <Flex.Box className="console-form" style={{ padding: "3rem" }} grow>
        <Form.Form<typeof Channel.calculatedFormSchema> {...form}>
          <Form.Field<string> path="name" label="Name">
            {(p) => (
              <Input.Text
                autoFocus
                level="h2"
                variant="text"
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
          <Flex.Box x>
            <Form.Field<string>
              path="dataType"
              label="Output Data Type"
              style={{ width: 150 }}
            >
              {({ variant: _, ...p }) => (
                <Telem.SelectDataType
                  {...p}
                  disabled={isIndex}
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
          </Flex.Box>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action={isEdit ? "Save" : "Create"} />
        <Nav.Bar.End align="center" gap="large">
          {isEdit && (
            <Flex.Box x align="center" gap="small">
              <Input.Switch value={createMore} onChange={setCreateMore} />
              <Text.Text color={9}>Create More</Text.Text>
            </Flex.Box>
          )}
          <Flex.Box x align="center">
            <Button.Button
              status={variant}
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
