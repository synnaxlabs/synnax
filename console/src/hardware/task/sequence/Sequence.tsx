// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/task/sequence/Sequence.css";

import { type channel, type rack } from "@synnaxlabs/client";
import {
  Channel,
  Flex,
  Form,
  Icon,
  type Input,
  Rack,
  Status,
  Synnax,
} from "@synnaxlabs/pluto";
import { unique } from "@synnaxlabs/x";
import { memo, useCallback, useEffect, useMemo, useRef } from "react";

import { Code } from "@/code";
import { Lua } from "@/code/lua";
import { usePhantomGlobals, type UsePhantomGlobalsReturn } from "@/code/phantom";
import { bindChannelsAsGlobals, useSuggestChannels } from "@/code/useSuggestChannels";
import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { GLOBALS } from "@/hardware/task/sequence/globals";
import {
  configZ,
  statusDetailsZ,
  TYPE,
  typeZ,
  ZERO_PAYLOAD,
} from "@/hardware/task/sequence/types";
import { type Modals } from "@/modals";
import { Selector } from "@/selector";

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

export const Selectable: Selector.Selectable = ({
  layoutKey,
  onPlace,
  rename,
  handleError,
}) => {
  const handleClick = useCallback(() => {
    handleError(async () => {
      const layout = await createLayout({ rename });
      if (layout != null) onPlace({ ...layout, key: layoutKey });
    }, "Failed to create Control Sequence");
  }, [onPlace, layoutKey, rename, handleError]);

  return (
    <Selector.Item
      key={TYPE}
      title="Control Sequence"
      icon={<Icon.Control />}
      onClick={handleClick}
    />
  );
};
Selectable.type = TYPE;

interface EditorProps extends Input.Control<string> {
  globals?: UsePhantomGlobalsReturn;
}

const Editor = memo(({ value, onChange, globals }: EditorProps) => {
  const methods = Form.useContext();
  const onAccept = useCallback(
    (channel: channel.Payload) => {
      globals?.set(channel.key.toString(), channel.name, channel.key.toString());
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
        chs.forEach((ch) =>
          globals?.set(ch.key.toString(), ch.name, ch.key.toString()),
        );
      })
      .catch(console.error);
  }, [methods, globals]);
  return <Code.Editor language={Lua.LANGUAGE} value={value} onChange={onChange} />;
});
Editor.displayName = "Editor";

const EditorField = Form.fieldBuilder<string, string, EditorProps>(Editor)({
  inputProps: {},
});

const Internal = ({
  layoutKey,
  onConfigure,
  status,
}: Common.Task.FormProps<typeof typeZ, typeof configZ, typeof statusDetailsZ>) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const globals = usePhantomGlobals({
    language: Lua.LANGUAGE,
    stringifyVar: Lua.stringifyVar,
    initialVars: GLOBALS,
  });
  const editorInputProps = useMemo(() => ({ globals }), [globals]);
  const initializedRef = useRef(false);
  if (status.variant === "success" && !initializedRef.current)
    initializedRef.current = true;

  return (
    <Flex.Box
      className={CSS.B("sequence")}
      style={{ padding: 0, height: "100%", minHeight: 0 }}
      y
      empty
    >
      {initializedRef.current && (
        <EditorField
          path="config.script"
          showLabel={false}
          showHelpText={false}
          padHelpText={false}
          grow
          inputProps={editorInputProps}
        />
      )}
      <Flex.Box
        pack
        y
        bordered={false}
        full="x"
        background={0}
        shrink={false}
        style={{
          boxShadow: "var(--pluto-shadow-v1)",
          borderTop: "var(--pluto-border)",
        }}
      >
        <Flex.Box y style={{ padding: "2rem", paddingBottom: "3rem" }} gap="medium">
          <Flex.Box x>
            <Form.Field<rack.Key>
              path="config.rack"
              label="Location"
              padHelpText={false}
              onChange={(v, { set }) => set("rackKey", v)}
              grow
            >
              {({ value, onChange }) => (
                <Rack.SelectSingle
                  allowNone={false}
                  value={value}
                  onChange={onChange}
                />
              )}
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
          </Flex.Box>
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
        </Flex.Box>
        <Common.Task.Controls
          layoutKey={layoutKey}
          formStatus={status}
          onConfigure={onConfigure}
        />
      </Flex.Box>
    </Flex.Box>
  );
};

const getInitialValues: Common.Task.GetInitialValues<
  typeof typeZ,
  typeof configZ,
  typeof statusDetailsZ
> = ({ config }) => {
  const cfg = config != null ? configZ.parse(config) : ZERO_PAYLOAD.config;
  return { ...ZERO_PAYLOAD, config: cfg };
};

const SCHEMAS = {
  type: typeZ,
  config: configZ,
  statusData: statusDetailsZ,
};

export const Sequence = Common.Task.wrapForm<
  typeof typeZ,
  typeof configZ,
  typeof statusDetailsZ
>({
  type: TYPE,
  Form: Internal,
  getInitialValues,
  schemas: SCHEMAS,
  onConfigure: async (_, config) => [config, config.rack],
  showHeader: false,
  showControls: false,
});
