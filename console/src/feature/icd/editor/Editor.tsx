// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/icd/editor/Editor.css";

import { icd, StaleError, type Synnax } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, ICD, Library, Panel } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Entries } from "@/feature/icd/editor/Entries";
import { Enum } from "@/feature/icd/editor/Enum";
import { Message } from "@/feature/icd/editor/Message";
import { Tasks } from "@/feature/icd/editor/Tasks";
import { useImport } from "@/feature/icd/useImport";
import { CSS } from "@/platform/css";

interface EntryProps {
  path: string;
}

const Entry = ({ path }: EntryProps): ReactElement => {
  const kind = Form.useFieldValue<icd.EntryType>(`${path}.kind`);
  return (
    <Flex.Box y gap="medium" className={CSS.BE("icd-editor", "entry")}>
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
  const { data } = Form.useFieldList<string, icd.Entry>("entries");
  // The selected entry can vanish when a save or an import replaces the entries.
  const current = selected != null && data.includes(selected) ? selected : data[0];
  return (
    <Flex.Box x grow empty className={CSS.BE("icd-editor", "body")}>
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

const check = async (client: Synnax, value: icd.ICD) => await client.icds.check(value);

interface StaleProps {
  onReload: () => void;
}

const Stale = ({ onReload }: StaleProps): ReactElement => (
  <Flex.Box x align="center" gap="small" grow>
    <Status.Summary
      variant="warning"
      message="This ICD changed after you opened it"
      description="Reload to get the changes. Reload discards your unsaved edits."
    />
    <Button.Button variant="outlined" onClick={onReload}>
      <Icon.Refresh />
      Reload
    </Button.Button>
  </Flex.Box>
);

interface ContentProps {
  onReload: () => void;
}

const Content = ({ onReload }: ContentProps): ReactElement => {
  const { key } = Panel.useTabResource();
  const [selected, setSelected] = useState<string | null>(null);
  const canEdit = Access.useUpdateGranted(icd.ontologyID(key));
  const { form, save, variant, status } = ICD.useForm({
    query: { key },
    initialValues: { key, name: "", entries: [], hash: "" },
    mode: canEdit ? "normal" : "preview",
  });
  Library.useCheck({ form, check });
  const stale = status.variant === "error" && StaleError.matches(status.details.error);
  const handleImport = useImport({ onImport: (imported) => form.reset(imported) });
  return (
    <Form.Form<typeof ICD.formSchema> {...form}>
      <Flex.Box y empty full className={CSS.B("icd-editor")}>
        {canEdit && (
          <Flex.Box
            x
            justify="end"
            gap="small"
            className={CSS.BE("icd-editor", "header")}
          >
            {stale && <Stale onReload={onReload} />}
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
        <Tasks icdKey={key} />
      </Flex.Box>
    </Form.Form>
  );
};

/**
 * Edits one ICD. Changes reach the Core when the user saves. A check marks each problem
 * shortly after an edit. A save of an ICD that changed elsewhere fails and offers a
 * reload.
 */
export const Editor = (): ReactElement => {
  const [generation, setGeneration] = useState(0);
  // A new key remounts the form, so it reads the ICD again.
  return <Content key={generation} onReload={() => setGeneration((g) => g + 1)} />;
};
