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
import { Text } from "@synnaxlabs/lyra/text";
import { Access, ICD, Panel } from "@synnaxlabs/pluto";
import { type deep } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Message } from "@/feature/icd/editor/Message";
import { Messages } from "@/feature/icd/editor/Messages";
import { useImport } from "@/feature/icd/useImport";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

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
}

const Body = ({ selected, onSelect }: BodyProps): ReactElement => {
  const { data } = Form.useFieldList(ICD.formPath.messages);
  // Selected messages can vanish when another client or an import removes them.
  const present = selected.filter((key) => data.includes(key));
  const shown = present.length > 0 ? present : data.slice(0, 1);
  const current = shown.at(0);
  const path = current == null ? null : ICD.formPath.messages[current];
  return (
    <Task.Views.Panes
      listTitle="Messages"
      list={<Messages selected={shown} onSelect={onSelect} />}
      detailsPath={path}
      title={path == null ? undefined : <Title path={path} />}
    >
      {path == null ? (
        <Flex.Box grow align="center" justify="center">
          <Text.Text color={9}>Add a message to start</Text.Text>
        </Flex.Box>
      ) : (
        <Flex.Box y grow empty className={CSS.BE("icd-editor", "message")}>
          <Message key={current} path={path} />
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
  const [selected, setSelected] = useState<string[]>([]);
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
        <Flex.Box x bordered grow empty className={CSS.BE("icd-editor", "body")}>
          <Body selected={selected} onSelect={setSelected} />
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
