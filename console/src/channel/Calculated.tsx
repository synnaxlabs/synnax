// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, framer } from "@synnaxlabs/client";
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
} from "@synnaxlabs/pluto";
import { deep, unique } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import * as monaco from "monaco-editor";
import { type ReactElement, useCallback, useEffect, useState } from "react";
import { z } from "zod";

import { baseFormSchema, createFormValidator, ZERO_CHANNEL } from "@/channel/Create";
import { Code } from "@/code";
import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Layout } from "@/layout";
import type { RendererProps } from "@/layout/slice";
import { Triggers } from "@/triggers";

export interface CalculatedLayoutArgs {
  channelKey?: number;
}

const DEFAULT_ARGS: CalculatedLayoutArgs = { channelKey: undefined };

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
  const addStatus = Status.useAdder();
  const handleException = Status.useExceptionHandler();
  Observe.useListener({
    key: [client?.key, addStatus, handleException],
    open: async () => {
      if (client == null) return;
      const s = await client.openStreamer({ channels: [CALCULATION_STATE_CHANNEL] });
      return new framer.ObservableStreamer(s);
    },
    onChange: (frame) => {
      const state = frame.get(CALCULATION_STATE_CHANNEL).parseJSON(calculationStateZ);
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
          .catch((e) => handleException(e, "Calculated channel failed"));
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

  const addStatus = Status.useAdder();

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

  const checkRequires = useMutation({
    mutationFn: async (fld: Form.FieldState<channel.Key[]>) => {
      const v = fld.value;
      if (client == null || v.length == 0) return;
      const channels = await client.channels.retrieve(v);
      const hyphenated = channels.filter((ch) => ch.name.includes("-"));
      if (!hyphenated.length) return;
      let base = "Channel ";
      if (hyphenated.length > 1) base = "Channels ";
      base += hyphenated.map((ch) => ch.name).join(", ");
      base += " with hyphens must be accessed using ";
      base += hyphenated.map((ch) => `get("${ch.name}")`).join(", ");
      if (hyphenated.length > 1) base += " as they are not valid variable names.";
      else base += " as it is not a valid variable name.";
      methods.setStatus("expression", {
        variant: "warning",
        message: base,
      });
    },
  });
  Form.useFieldListener<channel.Key[], typeof schema>({
    path: "requires",
    onChange: useCallback((v) => checkRequires.mutate(v), [checkRequires]),
    ctx: methods,
  });

  const autoFillRequires = useMutation({
    mutationFn: async ({
      value,
      extra,
    }: {
      value: string;
      extra: Form.ContextValue;
    }) => {
      if (client == null) return;
      const channelRegex = /\b([a-zA-Z][a-zA-Z0-9_-]*)\b/g;
      const requires = extra.get<channel.Key[]>("requires").value;
      const channelNames: string[] = [];
      let match: RegExpExecArray | null;
      while ((match = channelRegex.exec(value)) !== null) {
        const channelName = match[1];
        if (channelName) channelNames.push(channelName);
      }
      const channels = unique.by(
        await client.channels.retrieve(channelNames),
        ({ name }) => name,
      );
      if (channels.length == 0) return;
      const channelKeys = channels.map(({ key }) => key);
      extra.set("requires", unique.unique([...requires, ...channelKeys]));
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

          <Form.Field<string>
            path="expression"
            grow
            onChange={(v, extra) => autoFillRequires.mutate({ value: v, extra })}
          >
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
            >
              {({ variant: _, ...p }) => <Channel.SelectMultiple zIndex={100} {...p} />}
            </Form.Field>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Layout.BottomNavBar>
        <Triggers.SaveHelpText action={initialValues.key !== 0 ? "Save" : "Create"} />
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
              triggers={Triggers.SAVE}
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
  const ctx = Form.useContext();

  // Register Monaco editor commands and completion provider
  useEffect(() => {
    const disposables: monaco.IDisposable[] = [];
    disposables.push(
      monaco.editor.registerCommand("onSuggestionAccepted", (_, channelKey) =>
        ctx.set(
          "requires",
          unique.unique([...ctx.get<channel.Key[]>("requires").value, channelKey]),
        ),
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
          const channels = await client.channels.search(word.word, { internal: false });
          return {
            suggestions: channels.map((channel) => ({
              label: channel.name,
              kind: monaco.languages.CompletionItemKind.Variable,
              insertText: channel.name.includes("-")
                ? `get("${channel.name}")`
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

  return <Code.Editor {...props} />;
};
