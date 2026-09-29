// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/library/editor/Editor.css";

import { library } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Library, Panel } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Entries } from "@/feature/library/editor/Entries";
import { Enum } from "@/feature/library/editor/Enum";
import { Message } from "@/feature/library/editor/Message";
import { Tasks } from "@/feature/library/editor/Tasks";
import { useImport } from "@/feature/library/useImport";
import { CSS } from "@/platform/css";

interface EntryProps {
  path: string;
}

const Entry = ({ path }: EntryProps): ReactElement => {
  const kind = Form.useFieldValue<library.EntryType>(`${path}.kind`);
  return (
    <Flex.Box y gap="medium" className={CSS.BE("library-editor", "entry")}>
      <Form.TextField path={`${path}.name`} label="Name" />
      {kind === "enum" ? <Enum path={path} /> : <Message path={path} />}
    </Flex.Box>
  );
};

interface BodyProps {
  selected: string | null;
  onSelect: (key: string | null) => void;
}

const Body = ({ selected, onSelect }: BodyProps): ReactElement => {
  const { data } = Form.useFieldList<string, library.Entry>("entries");
  // The selected entry can vanish when a save or an import replaces the entries.
  const current = selected != null && data.includes(selected) ? selected : data[0];
  return (
    <Flex.Box x grow empty className={CSS.BE("library-editor", "body")}>
      <Entries selected={current ?? null} onSelect={onSelect} />
      {current == null ? (
        <Flex.Box grow align="center" justify="center">
          <Text.Text color={9}>Add an enum or a message to start</Text.Text>
        </Flex.Box>
      ) : (
        <Entry key={current} path={`entries.${current}`} />
      )}
    </Flex.Box>
  );
};

/** Edits one library. Changes reach the Core when the user saves. */
export const Editor = (): ReactElement => {
  const { key } = Panel.useTabResource();
  const [selected, setSelected] = useState<string | null>(null);
  const canEdit = Access.useUpdateGranted(library.ontologyID(key));
  const { form, save, variant } = Library.useForm({
    query: { key },
    initialValues: { key, name: "", entries: [] },
    mode: canEdit ? "normal" : "preview",
  });
  const handleImport = useImport({ onImport: (imported) => form.reset(imported) });
  return (
    <Form.Form<typeof Library.formSchema> {...form}>
      <Flex.Box y empty full className={CSS.B("library-editor")}>
        {canEdit && (
          <Flex.Box
            x
            justify="end"
            gap="small"
            className={CSS.BE("library-editor", "header")}
          >
            <Button.Button variant="outlined" onClick={() => handleImport(key)}>
              <Icon.Import />
              Import
            </Button.Button>
            <Button.Button
              variant="filled"
              onClick={() => save()}
              status={variant === "loading" ? "loading" : undefined}
            >
              <Icon.Save />
              Save
            </Button.Button>
          </Flex.Box>
        )}
        <Body selected={selected} onSelect={setSelected} />
        <Tasks libraryKey={key} />
      </Flex.Box>
    </Form.Form>
  );
};
