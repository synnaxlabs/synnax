// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, Rate } from "@synnaxlabs/client";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import {
  Align,
  Button,
  Channel,
  Form,
  Input,
  Nav,
  Select,
  Synnax,
  Text,
  Triggers,
  useSyncedRef,
  Status,
} from "@synnaxlabs/pluto";
import { unique, deep } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import * as monaco from "monaco-editor";
import { type ReactElement, useEffect, useState } from "react";
import { z } from "zod";

import { Code } from "@/code";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import type { RendererProps } from "@/layout/slice";
import { useQuery } from "@tanstack/react-query";

export interface CalculatedChannelArgs {
  channelKey?: number;
}

const defaultArgs: CalculatedChannelArgs = {
  channelKey: undefined
};



const schema = channel.newPayload
  .extend({
    name: z.string().min(1, "Name must not be empty"),
    dataType: DataType.z.transform((v) => v.toString()),
  })
  .refine((v) => !v.isIndex || new DataType(v.dataType).equals(DataType.TIMESTAMP), {
    message: "Index channel must have data type TIMESTAMP",
    path: ["dataType"],
  })
  .refine((v) => v.isIndex || v.index !== 0 || v.virtual, {
    message: "Data channel must have an index",
    path: ["index"],
  });

type FormValues = z.infer<typeof schema>;

export const CREATE_CALCULATED_LAYOUT_TYPE = "createCalculatedChannel";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const createCalculatedLayout: Layout.State = {
  key: CREATE_CALCULATED_LAYOUT_TYPE,
  type: CREATE_CALCULATED_LAYOUT_TYPE,
  windowKey: MAIN_WINDOW,
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
};

const ZERO_FORM_VALUES: FormValues = {
  key: 0,
  name: "",
  index: 0,
  dataType: "float32",
  isIndex: false,
  leaseholder: 0,
  virtual: true,  // Set to true by default
  rate: Rate.hz(0),
  internal: false,
  expression: "np.array([])",
  requires: [],
};

export const CreateCalculatedModal: Layout.Renderer = ({ layoutKey, onClose }) => {
  const client = Synnax.use();
  const args = Layout.useSelectArgs<CalculatedChannelArgs>(layoutKey) ?? defaultArgs
  const res = useQuery<FormValues>({
    queryKey: [args.channelKey, client?.key],
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
  if (res.isSuccess) return <Internal onClose={onClose} initialValues={res.data} />;
  return null;
};

interface InternalProps extends Pick<RendererProps, "onClose"> {
  initialValues: FormValues;
}

const Internal = ({ onClose, initialValues }: InternalProps): ReactElement => {
  const client = Synnax.use();

  const methods = Form.use<typeof schema>({ schema, values: initialValues });

  const [createMore, setCreateMore] = useState(false);
  const { mutate, isPending } = useMutation({
    mutationFn: async (createMore: boolean) => {
      const result = methods.validate();
      if (!result) {
        throw new Error("Form validation failed");
      }

      if (client == null) throw new Error("Client not available");
      const formData = methods.value();
      await client.channels.create(formData);
      if (!createMore) onClose();
      else methods.reset(deep.copy(ZERO_FORM_VALUES));
    },
    onError: (error: Error) => {
      console.error("Mutation error:", error.message);
    }
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
          <Align.Space direction="x" size="large">
            <Form.Field<DataType> path="dataType" label="Output Data Type" grow>
              {(p) => (
                <Select.DataType
                  {...p}
                  disabled={isIndex}
                  maxHeight="small"
                  zIndex={100}
                />
              )}
            </Form.Field>
          </Align.Space>

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
          <Form.Field<channel.Key[]> path="requires" label="Required Channels" grow>
            {(p) => <Channel.SelectMultiple zIndex={100} {...p} />}
          </Form.Field>
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
          <Button.Button
            disabled={isPending}
            loading={isPending}
            onClick={() => mutate(createMore)}
            triggers={[SAVE_TRIGGER]}
          >
            {initialValues.key !== 0 ? "Update Channel" : "Create Channel"}
          </Button.Button>
        </Nav.Bar.End>
      </Layout.BottomNavBar>
    </Align.Space>
  );
};

const Editor = (props: Code.EditorProps): ReactElement => {
  const client = Synnax.use();
  const requires = Form.useField<channel.Key[]>({ path: "requires" });
  const valueRef = useSyncedRef(requires.value);

  useEffect(() => {
    const disposables: monaco.IDisposable[] = [];
    disposables.push(
      monaco.editor.registerCommand("onSuggestionAccepted", (_, channelKey) =>
        requires.onChange(unique([...valueRef.current, channelKey])),
      ),
    );

    disposables.push(
      monaco.languages.registerCompletionItemProvider("python", {
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
              insertText: channel.name,
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
