// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, framer } from "@synnaxlabs/client";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
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
  Triggers,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { deep, unique } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import * as monaco from "monaco-editor";
import { type ReactElement, useEffect, useState } from "react";
import { z } from "zod";

import { baseFormSchema, createFormValidator, ZERO_CHANNEL } from "@/channel/Create";
import { Code } from "@/code";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modal } from "@/layout/Modal";
import type { RendererProps } from "@/layout/slice";
import { Version } from "@/version";

export interface CalculatedChannelArgs {
  channelKey?: number;
}

const DEFAULT_ARGS: CalculatedChannelArgs = { channelKey: undefined };

const schema = createFormValidator(
  baseFormSchema
    .extend({
      name: z.string().min(1, "Name must not be empty"),
      dataType: DataType.z.transform((v) => v.toString()),
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
    }),
);

type FormValues = z.infer<typeof schema>;

export const CREATE_CALCULATED_LAYOUT_TYPE = "createCalculatedChannel";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const createCalculatedLayout = (base: Partial<Layout.State>): Layout.State => ({
  betaFeature: {
    name: "Calculated channels",
    plural: true,
  },
  name: "Channel.Create.Calculated",
  icon: "Channel",
  location: "modal",
  tab: {
    closable: true,
    editable: false,
  },
  window: {
    resizable: false,
    size: { height: 600, width: 1000 },
    navTop: true,
    showTitle: true,
  },
  ...base,
  key: CREATE_CALCULATED_LAYOUT_TYPE,
  type: CREATE_CALCULATED_LAYOUT_TYPE,
  windowKey: MAIN_WINDOW,
});

const ZERO_FORM_VALUES: FormValues = {
  ...ZERO_CHANNEL,
  virtual: true, // Set to true by default
  expression: "return 0",
};

const calculationStateZ = z.object({
  key: channel.keyZ,
  variant: z.enum(["error", "success", "info"]),
  message: z.string(),
});

const CALCULATION_STATE_CHANNEL = "sy_calculation_state";

export const useListenForCalculationState = (): void => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  Observe.useListener({
    key: [CALCULATION_STATE_CHANNEL, client?.key],
    open: async () => {
      if (client == null) return;
      const s = await client.openStreamer({ channels: [CALCULATION_STATE_CHANNEL] });
      return new framer.ObservableStreamer(s);
    },
    onChange: (frame) => {
      const state = frame.get(CALCULATION_STATE_CHANNEL).parseJSON(calculationStateZ);
      state.forEach(({ key, variant, message }) => {
        client?.channels.retrieve(key).then((ch) => {
          addStatus({
            variant,
            message: `Calculation for ${ch.name} failed`,
            description: message,
          });
        });
      });
    },
  });
};

export const CreateCalculatedModal: Layout.Renderer = ({ layoutKey, onClose }) => {
  const client = Synnax.use();
  const args = Layout.useSelectArgs<CalculatedChannelArgs>(layoutKey) ?? DEFAULT_ARGS;
  const res = useQuery<FormValues>({
    queryKey: [args.channelKey, client?.key],
    staleTime: 0,
    queryFn: async () => {
      if (args.channelKey == null) return deep.copy(ZERO_FORM_VALUES);
      if (client == null) throw new Error("Client not available");
      const ch = await client.channels.retrieve(args.channelKey);
      return { ...ch, dataType: ch.dataType.toString() };
    },
  });

  if (res.isLoading) return <Text.Text level="p">Loading...</Text.Text>;
  if (res.isError)
    return (
      <Align.Space direction="y" grow style={{ height: "100%" }}>
        <Status.Text.Centered variant="error">{res.error.message}</Status.Text.Centered>
      </Align.Space>
    );

  return <Internal onClose={onClose} initialValues={res.data} />;
};

interface InternalProps extends Pick<RendererProps, "onClose"> {
  initialValues: FormValues;
}

const Internal = ({ onClose, initialValues }: InternalProps): ReactElement => {
  const client = Synnax.use();

  const methods = Form.use<typeof schema>({
    schema,
    values: initialValues,
    sync: true,
  });

  const addStatus = Status.useAggregator();

  const [createMore, setCreateMore] = useState(false);
  const { mutate, isPending } = useMutation({
    mutationFn: async (createMore: boolean) => {
      if (client == null) throw new Error("Client not available");
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

  return (
    <Align.Space className={CSS.B("channel-edit-layout")} grow empty>
      <Align.Space className="console-form" style={{ padding: "3rem" }} grow>
        <Form.Form {...methods}>
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
                lang="python"
                onChange={onChange}
                bordered
                rounded
                style={{ height: 150 }}
              />
            )}
          </Form.Field>
          <Align.Space direction="x">
            <Form.Field<DataType>
              path="dataType"
              label="Output Data Type"
              style={{ width: 150 }}
            >
              {(p) => (
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
            >
              {(p) => <Channel.SelectMultiple zIndex={100} {...p} />}
            </Form.Field>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Layout.BottomNavBar>
        <Nav.Bar.Start size="small">
          <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
          <Text.Text shade={7} level="small">
            To Save
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End align="center" size="large">
          {initialValues.key !== 0 && (
            <Align.Space direction="x" align="center" size="small">
              <Input.Switch value={createMore} onChange={setCreateMore} />
              <Text.Text level="p" shade={7}>
                Create More
              </Text.Text>
            </Align.Space>
          )}
          <Align.Space direction="x" align="center">
            <Button.Button
              disabled={isPending}
              loading={isPending}
              onClick={() => mutate(createMore)}
              triggers={[SAVE_TRIGGER]}
            >
              {initialValues.key !== 0 ? "Save" : "Create"}
            </Button.Button>
          </Align.Space>
        </Nav.Bar.End>
      </Layout.BottomNavBar>
    </Align.Space>
  );
};

const Editor = (props: Code.EditorProps): ReactElement => {
  const client = Synnax.use();
  const requires = Form.useField<channel.Key[]>({ path: "requires" });
  const valueRef = useSyncedRef(requires.value);
  const [hyphenWarning, setHyphenWarning] = useState<string | null>(null);

  const hasHyphenatedName = async () => {
    if (!client || !requires.value?.length) return false;
    const channels = await Promise.all(
      requires.value.map((key) => client.channels.retrieve(key)),
    );
    return channels.some((ch) => ch.name.includes("-"));
  };

  useEffect(() => {
    const checkHyphens = async () => {
      const hasHyphen = await hasHyphenatedName();
      if (hasHyphen)
        setHyphenWarning(
          "Note: Channels with hyphens must be accessed using" +
            ' channels["channel-name"]',
        );
      else setHyphenWarning(null);
    };
    checkHyphens();
  }, [requires.value]);

  // Specifically to handle generating requires list when reopening existing calc channel
  useEffect(() => {
    if (!client || !props.value) return;
    const updateRequiredChannels = async () => {
      try {
        const channelRegex = /\b([a-zA-Z][a-zA-Z0-9_-]*)\b/g;
        const channelNames: string[] = [];
        let match;

        // Extract all matches
        while ((match = channelRegex.exec(props.value)) !== null) {
          const channelName = match[1];
          if (channelName) channelNames.push(channelName);
        }

        const channels = await Promise.all(
          channelNames.map(async (name) => {
            const results = await client.channels.search(name, { internal: false });
            const exactMatch = results.find((ch) => ch.name === name);
            return exactMatch;
          }),
        );
        const channelKeys = channels
          .filter((ch): ch is NonNullable<typeof ch> => ch != null)
          .map((ch) => ch.key);

        if (channelKeys.length > 0) {
          const newRequires = unique.unique([...valueRef.current, ...channelKeys]);
          requires.onChange(newRequires);
        }
      } catch (error) {
        console.error("Error updating required channels:", error);
      }
    };
    updateRequiredChannels();
  }, [client, props.value]);

  // Register Monaco editor commands and completion provider
  useEffect(() => {
    const disposables: monaco.IDisposable[] = [];
    disposables.push(
      monaco.editor.registerCommand("onSuggestionAccepted", (_, channelKey) =>
        requires.onChange(unique.unique([...valueRef.current, channelKey])),
      ),
    );

    disposables.push(
      monaco.languages.registerCompletionItemProvider("lua", {
        triggerCharacters: ["."],
        provideCompletionItems: async (
          model: monaco.editor.ITextModel,
          position: monaco.Position,
        ): Promise<monaco.languages.CompletionList> => {
          if (client == null) return { suggestions: [] };
          const word = model.getWordUntilPosition(position);
          const range: monaco.IRange = {
            startLineNumber: position.lineNumber,
            endLineNumber: position.lineNumber,
            startColumn: word.startColumn,
            endColumn: word.endColumn,
          };
          const channels = await client?.channels.search(word.word, {
            internal: false,
          });
          return {
            suggestions: channels.map((channel) => ({
              label: channel.name,
              kind: monaco.languages.CompletionItemKind.Variable,
              insertText: channel.name.includes("-")
                ? `channels["${channel.name}"]`
                : channel.name,
              range,
              command: {
                id: "onSuggestionAccepted",
                title: "Suggestion Accepted",
                arguments: [channel.key],
              },
            })),
          };
        },
      }),
    );
    return () => disposables.forEach((d) => d.dispose());
  }, []);

  return (
    <>
      <Code.Editor {...props} />
      {hyphenWarning && (
        <Text.Text level="small" shade={7}>
          {hyphenWarning}
        </Text.Text>
      )}
    </>
  );
};
