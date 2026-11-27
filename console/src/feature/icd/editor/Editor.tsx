// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/icd/editor/Editor.css";

import { icd } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Portal } from "@synnaxlabs/lyra/portal";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, ICD, Panel } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { Framing } from "@/feature/icd/editor/Framing";
import { Message } from "@/feature/icd/editor/Message";
import { Messages } from "@/feature/icd/editor/Messages";
import { useImport } from "@/feature/icd/useImport";
import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Task } from "@/platform/task";
import { Session } from "@/session";

const uniqueName = (messages: icd.Message[]): string => {
  const names = new Set(messages.map(({ name }) => name));
  let i = messages.length + 1;
  while (names.has(`Message ${i}`)) i++;
  return `Message ${i}`;
};

const createMessage = (messages: icd.Message[]): icd.Message => ({
  key: uuid.create(),
  name: uniqueName(messages),
  payload: { format: "binary", fields: [] },
});

interface TitleProps {
  path: deep.Path<icd.Message>;
}

const Title = ({ path }: TitleProps): ReactElement | null => {
  const name = Form.useFieldValue(path.name, { optional: true });
  return (
    <Text.Text weight={500} color={10} overflow="fade">
      {name}
    </Text.Text>
  );
};

interface BodyProps {
  selected: string[];
  onSelect: (keys: string[]) => void;
  /** A field of the shown message to focus. */
  focused?: icd.FieldKey;
  /** Called once the body tried to focus the field. */
  onFocused: () => void;
}

const Body = ({ selected, onSelect, focused, onFocused }: BodyProps): ReactElement => {
  const { focus, mode } = Form.useContext();
  const messages = Form.useFieldList(ICD.formPath.messages);
  const { data } = messages;
  // The message that the user just added, whose name takes focus once it shows.
  const [added, setAdded] = useState<icd.MessageKey | null>(null);
  const handleAdd = () => {
    const message = createMessage(messages.value());
    messages.push(message);
    setAdded(message.key);
    onSelect([message.key]);
  };
  const handleSelect = (keys: string[]) => {
    setAdded(null);
    onSelect(keys);
  };
  // Selected messages can vanish when another client or an import removes them.
  const present = selected.filter((key) => data.includes(key));
  const shown = present.length > 0 ? present : data.slice(0, 1);
  const current = shown.at(0);
  const path = current == null ? null : ICD.formPath.messages[current];
  // The fields of a message mount with it, so they take focus once it shows. Another
  // client can remove the field first.
  useEffect(() => {
    if (focused == null || current == null) return;
    const fields = deep.widen<unknown>(ICD.formPath.messages[current]).payload.fields;
    focus(fields[focused].name, { optional: true });
    onFocused();
  }, [focused, current]);
  return (
    <Task.Views.Panes
      listTitle="Messages"
      list={<Messages selected={shown} onSelect={handleSelect} onAdd={handleAdd} />}
      detailsPath={path}
      title={path == null ? undefined : <Title path={path} />}
    >
      {path == null ? (
        <Empty.Action
          grow
          message="No message selected"
          action={mode === "preview" ? undefined : "Add message"}
          onClick={handleAdd}
        />
      ) : (
        <Flex.Box y grow empty className={CSS.BE("icd-editor", "message")}>
          <Message key={current} path={path} autoFocus={current === added} />
        </Flex.Box>
      )}
    </Task.Views.Panes>
  );
};

/**
 * Edits one ICD. Each edit reaches the Core at once, and edits from other clients show
 * as they arrive. A check marks each problem shortly after an edit.
 */
export const Editor = (): ReactElement => {
  const { key } = Panel.useTabResource();
  const dispatch = Session.useDispatch();
  const selected = Session.ICD.useSelectSelected(key);
  const focused = Session.ICD.useSelectFocused(key);
  const created = Session.ICD.useSelectCreated(key);
  const attached = Portal.useAttached();
  const handleSelect = (selected: icd.MessageKey[]) =>
    dispatch(Session.ICD.setSelected({ key, selected }));
  const handleFocused = () => dispatch(Session.ICD.clearFocused({ key }));
  // The name took focus as it mounted.
  useEffect(() => {
    if (created) dispatch(Session.ICD.setCreated({ key, created: false }));
  }, [created]);
  const canEdit = Access.useUpdateGranted(icd.resourceID(key));
  const { form } = ICD.useForm({
    query: { key },
    mode: canEdit ? "normal" : "preview",
  });
  ICD.useCheck({ key, form });
  const handleImport = useImport();
  return (
    <Form.Form<typeof ICD.formSchema> {...form}>
      <Flex.Box y empty full className={CSS.B("icd-editor")}>
        <Flex.Box
          x
          align="center"
          justify="between"
          gap="medium"
          className={CSS.BE("icd-editor", "header")}
        >
          <Form.Field
            path={ICD.formPath.name}
            showLabel={false}
            padHelpText={false}
            className={CSS.BE("icd-editor", "name")}
          >
            {(p) => (
              <Input.Text
                variant="text"
                level="h3"
                onlyChangeOnBlur
                aria-label="Name"
                autoFocus={created}
                selectOnFocus={created}
                {...p}
              />
            )}
          </Form.Field>
          {canEdit && (
            <Button.Button variant="outlined" onClick={() => handleImport(key)}>
              <Icon.Import />
              Import
            </Button.Button>
          )}
        </Flex.Box>
        {FLAGS.bus && <Framing />}
        <Flex.Box x bordered grow empty className={CSS.BE("icd-editor", "body")}>
          <Body
            selected={selected}
            onSelect={handleSelect}
            // Detached content cannot take focus, so the request waits for the tab.
            focused={attached ? focused : undefined}
            onFocused={handleFocused}
          />
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
