// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, rack, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Channel,
  Form,
  type Input,
  Rack,
  Status,
  Synnax,
} from "@synnaxlabs/pluto";
import { unique } from "@synnaxlabs/x";
import { useCallback, useEffect } from "react";

import { Code } from "@/code";
import { Lua } from "@/code/lua";
import { usePhantomGlobals, type UsePhantomGlobalsReturn } from "@/code/phantom";
import { bindChannelsAsGlobals, useSuggestChannels } from "@/code/useSuggestChannels";
import { Common } from "@/hardware/common";
import { Controls } from "@/hardware/common/task/Controls";
import { type Schema, useForm } from "@/hardware/common/task/Form";
import { GLOBALS } from "@/hardware/task/sequence/globals";
import {
  type Config,
  configZ,
  type StateDetails,
  TYPE,
  type Type,
  ZERO_PAYLOAD,
} from "@/hardware/task/sequence/types";
import { type Modals } from "@/modals";
import { type Selector } from "@/selector";

const FAILED_TO_UPDATE_AUTOCOMPLETE = "Failed to update sequence auto-complete";

export const LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  icon: "Control",
  name: ZERO_PAYLOAD.name,
  type: TYPE,
};

export interface CreateLayoutArgs {
  rackKey?: rack.Key;
  rename: Modals.PromptRename;
}

export const createLayout = async ({
  rackKey,
  rename,
}: CreateLayoutArgs): Promise<Common.Task.Layout | null> => {
  const name = await rename({}, { icon: "Control", name: "Control.Sequence.Create" });
  return name == null ? null : { ...LAYOUT, name, args: { rackKey } };
};

export const SELECTABLE: Selector.Selectable = {
  key: TYPE,
  title: "Control Sequence",
  icon: <Icon.Control />,
  create: async ({ layoutKey, rename }) => {
    const layout = await createLayout({ rename });
    return layout == null ? null : { ...layout, key: layoutKey };
  },
};

interface EditorProps extends Input.Control<string> {
  globals: UsePhantomGlobalsReturn;
}

const Editor = ({ value, onChange, globals }: EditorProps) => {
  const methods = Form.useContext();
  const onAccept = useCallback(
    (channel: channel.Payload) => {
      globals.set(channel.key.toString(), channel.name, channel.key.toString());
      methods.set(
        "config.read",
        unique.unique([
          ...methods.get<channel.Key[]>("config.read").value,
          channel.key,
        ]),
      );
    },
    [methods, globals],
  );
  useSuggestChannels(onAccept);
  const client = Synnax.use();
  useEffect(() => {
    const channels = methods.get<channel.Key[]>("config.read").value;
    client?.channels
      .retrieve(channels)
      .then((chs) => {
        chs.forEach((ch) => globals.set(ch.key.toString(), ch.name, ch.key.toString()));
      })
      .catch(console.error);
  }, [methods, globals]);
  return <Code.Editor language={Lua.LANGUAGE} value={value} onChange={onChange} />;
};

const schema = configZ.extend({
  rack: rack.keyZ.min(1, "Location is required"),
});

const Internal = ({
  task: base,
  layoutKey,
  rackKey,
}: Common.Task.TaskProps<Config, StateDetails, Type>) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const { formProps, handleConfigure, handleStartOrStop, state, isConfiguring } =
    useForm({
      task: {
        ...base,
        config: {
          ...base.config,
          rack: rackKey ?? task.getRackKey(base.key ?? "0"),
        },
      },
      layoutKey,
      configSchema: schema,
      type: TYPE,
      onConfigure: async (_, config) => [config, config.rack],
    });
  const { configured, isSnapshot, methods } = formProps;

  const globals = usePhantomGlobals({
    language: Lua.LANGUAGE,
    stringifyVar: Lua.stringifyVar,
    initialVars: GLOBALS,
  });

  return (
    <Align.Space style={{ padding: 0, height: "100%", minHeight: 0 }} y empty>
      <Form.Form<Schema<Config>> {...methods}>
        <Form.Field<string>
          path="config.script"
          showLabel={false}
          showHelpText={false}
          padHelpText={false}
          grow
        >
          {(p) => <Editor {...p} globals={globals} />}
        </Form.Field>
        <Align.Pack
          y
          bordered={false}
          style={{
            width: "100%",
            background: "var(--pluto-gray-l0)",
            boxShadow: "var(--pluto-shadow-v1)",
            borderTop: "var(--pluto-border)",
            flexShrink: 0, // Prevent the bottom section from shrinking
          }}
        >
          <Align.Space
            y
            style={{ padding: "2rem", paddingBottom: "3rem" }}
            size="medium"
          >
            <Align.Space x>
              <Form.Field<rack.Key>
                path="config.rack"
                label="Location"
                padHelpText={false}
                grow
              >
                {(p) => <Rack.SelectSingle allowNone={false} {...p} />}
              </Form.Field>
              <Form.NumericField
                label="Loop Rate"
                path="config.rate"
                padHelpText={false}
                style={{ width: 120 }}
                inputProps={{
                  endContent: "Hz",
                  bounds: { lower: 1, upper: 1001 },
                  dragScale: { x: 1, y: 1 },
                }}
              />
            </Align.Space>
            <Form.Field<channel.Key[]>
              path="config.read"
              label="Read From"
              padHelpText={false}
              onChange={(v, extra) => {
                if (client == null) return;
                handleError(
                  async () =>
                    await bindChannelsAsGlobals(
                      client,
                      extra.get<channel.Key[]>("config.read").value,
                      v,
                      globals,
                    ),
                  FAILED_TO_UPDATE_AUTOCOMPLETE,
                );
              }}
            >
              {({ value, onChange }) => (
                <Channel.SelectMultiple
                  value={value}
                  onChange={onChange}
                  location="top"
                />
              )}
            </Form.Field>
            <Form.Field<channel.Key[]>
              path="config.write"
              label="Write To"
              padHelpText={false}
              onChange={(v, extra) => {
                if (client == null) return;
                handleError(
                  async () =>
                    await bindChannelsAsGlobals(
                      client,
                      extra.get<channel.Key[]>("config.write").value,
                      v,
                      globals,
                    ),
                  FAILED_TO_UPDATE_AUTOCOMPLETE,
                );
              }}
            >
              {({ value, onChange }) => (
                <Channel.SelectMultiple
                  value={value}
                  onChange={onChange}
                  location="top"
                />
              )}
            </Form.Field>
          </Align.Space>
          <Controls
            layoutKey={layoutKey}
            state={state}
            isConfiguring={isConfiguring}
            onStartStop={handleStartOrStop}
            onConfigure={handleConfigure}
            isSnapshot={isSnapshot}
            hasBeenConfigured={configured}
            style={{
              padding: "2rem",
              border: "none",
              borderTop: "var(--pluto-border)",
            }}
          />
        </Align.Pack>
      </Form.Form>
    </Align.Space>
  );
};

export const Sequence = Common.Task.wrap(Internal, {
  getInitialPayload: () => ZERO_PAYLOAD,
  configSchema: configZ,
});
