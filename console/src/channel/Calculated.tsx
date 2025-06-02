// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, framer } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Channel,
  Form,
  Input,
  Nav,
  Observe,
  Select,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { deep, unique } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { baseFormSchema, ZERO_CHANNEL } from "@/channel/Create";
import { Code } from "@/code";
import { Lua } from "@/code/lua";
import { usePhantomGlobals, type UsePhantomGlobalsReturn } from "@/code/phantom";
import { bindChannelsAsGlobals, useSuggestChannels } from "@/code/useSuggestChannels";
import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

const FAILED_TO_UPDATE_AUTOCOMPLETE =
  "Failed to update calculated channel auto-complete";

export interface CalculatedLayoutArgs {
  channelKey?: number;
}

const DEFAULT_ARGS: CalculatedLayoutArgs = { channelKey: undefined };

const schema = baseFormSchema
  .extend({
    expression: z
      .string()
      .min(1, "Expression must not be empty")
      .refine((v) => v.includes("return"), {
        message: "Expression must contain a return statement",
      }),
  })
  .refine((v) => v.requires?.length > 0, {
    message: "Expression must use at least one synnax channel",
    path: ["requires"],
  });

type FormValues = z.output<typeof schema>;

export const CALCULATED_LAYOUT_TYPE = "createCalculatedChannel";

export interface CalculatedLayout extends Layout.BaseState<CalculatedLayoutArgs> {}

export const CALCULATED_LAYOUT: CalculatedLayout = {
  beta: true,
  name: "Channel.Create.Calculated",
  icon: "Channel",
  location: "modal",
  tab: { closable: true, editable: false },
  window: {
    resizable: false,
    size: { height: 600, width: 1000 },
    navTop: true,
    showTitle: true,
  },
  type: CALCULATED_LAYOUT_TYPE,
  key: CALCULATED_LAYOUT_TYPE,
};

export interface CreateCalculatedLayoutArgs {
  key: channel.Key;
  name: channel.Name;
}

export const createCalculatedLayout = ({
  key,
  name,
}: CreateCalculatedLayoutArgs): CalculatedLayout => ({
  ...CALCULATED_LAYOUT,
  args: { channelKey: key },
  name: `${name}.Edit`,
});

const ZERO_FORM_VALUES: FormValues = {
  ...ZERO_CHANNEL,
  virtual: true,
  expression: "return 0",
};

export const useListenForCalculationState = (): void => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  Observe.useListener({
    key: [client?.key, addStatus, handleError],
    open: async () => {
      if (client == null) return;
      const s = await client.openStreamer({
        channels: [channel.CALCULATION_STATE_CHANNEL_NAME],
      });
      return new framer.ObservableStreamer(s);
    },
    onChange: (frame) => {
      const state = frame
        .get(channel.CALCULATION_STATE_CHANNEL_NAME)
        .parseJSON(channel.calculationStateZ);
      state.forEach(({ key, variant, message }) => {
        client?.channels
          .retrieve(key)
          .then((ch) => {
            if (variant !== "error") {
              addStatus({ variant, message });
              return;
            }
            addStatus({
              variant,
              message: `Calculation for ${ch.name} failed`,
              description: message,
            });
          })
          .catch((e) => handleError(e, "Calculated channel failed"));
      });
    },
  });
};

export const Calculated: Layout.Renderer = ({ layoutKey, onClose }) => {
  const client = Synnax.use();
  const args = Layout.useSelectArgs<CalculatedLayoutArgs>(layoutKey) ?? DEFAULT_ARGS;
  const res = useQuery<FormValues>({
    queryKey: [args.channelKey, client?.key],
    staleTime: 0,
    queryFn: async () => {
      if (args.channelKey == null) return deep.copy(ZERO_FORM_VALUES);
      if (client == null) throw NULL_CLIENT_ERROR;
      const ch = await client.channels.retrieve(args.channelKey);
      return { ...ch.payload, dataType: ch.dataType.toString() };
    },
  });

  if (res.isLoading) return <Text.Text level="p">Loading...</Text.Text>;
  if (res.isError)
    return (
      <Align.Space y grow style={{ height: "100%" }}>
        <Status.Text.Centered variant="error">{res.error.message}</Status.Text.Centered>
      </Align.Space>
    );

  return <Internal onClose={onClose} initialValues={res.data as FormValues} />;
};

interface InternalProps extends Pick<Layout.RendererProps, "onClose"> {
  initialValues: FormValues;
}

const Internal = ({ onClose, initialValues }: InternalProps): ReactElement => {
  const client = Synnax.use();

  const methods = Form.use<typeof schema>({
    schema,
    values: initialValues,
    sync: true,
  });

  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const [createMore, setCreateMore] = useState(false);
  const { mutate, isPending } = useMutation({
    mutationFn: async (createMore: boolean) => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (!methods.validate()) return;
      const d = methods.value();
      await client.channels.create(d);
      if (!createMore) onClose();
      else methods.reset(deep.copy(ZERO_FORM_VALUES));
    },
    onError: (error: Error) => {
      addStatus({
        variant: "error",
        message: "Error creating calculated channel: ".concat(methods.value().name),
        description: error.message,
      });
    },
  });

  const isIndex = Form.useFieldValue<boolean, boolean, typeof schema>(
    "isIndex",
    false,
    methods,
  );

  const globals = usePhantomGlobals({
    language: Lua.LANGUAGE,
    stringifyVar: Lua.stringifyVar,
  });
  useAsyncEffect(async () => {
    if (client == null) return;
    const channels = methods.get<channel.Key[]>("requires").value;
    try {
      const chs = await client.channels.retrieve(channels);
      chs.forEach((ch) => globals.set(ch.key.toString(), ch.name, ch.key.toString()));
    } catch (e) {
      handleError(e, FAILED_TO_UPDATE_AUTOCOMPLETE);
    }
  }, [methods, globals, client]);

  return (
    <Align.Space className={CSS.B("channel-edit-layout")} grow empty>
      <Align.Space className="console-form" style={{ padding: "3rem" }} grow>
        <Form.Form<typeof schema> {...methods}>
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
        <Triggers.SaveHelpText action={initialValues.key !== 0 ? "Save" : "Create"} />
        <Nav.Bar.End align="center" size="large">
          {initialValues.key !== 0 && (
            <Align.Space x align="center" size="small">
              <Input.Switch value={createMore} onChange={setCreateMore} />
              <Text.Text level="p" shade={11}>
                Create More
              </Text.Text>
            </Align.Space>
          )}
          <Align.Space x align="center">
            <Button.Button
              disabled={isPending}
              loading={isPending}
              onClick={() => mutate(createMore)}
              triggers={Triggers.SAVE}
            >
              {initialValues.key !== 0 ? "Save" : "Create"}
            </Button.Button>
          </Align.Space>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Align.Space>
  );
};

export interface EditorProps extends Code.EditorProps {
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
