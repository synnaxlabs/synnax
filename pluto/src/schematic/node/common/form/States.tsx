// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import type { deep } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Access } from "@/access";
import { Enumeration } from "@/enumeration";

const NONE = "none";

export interface EnumerationSelectProps {
  /** The enumeration that the select names. */
  value?: enumeration.Key;
  onChange: (next?: enumeration.Key) => void;
  /** Returns the name and entries that a new enumeration starts with. */
  draft: () => Enumeration.CreatePromptParams;
  /**
   * The enumeration that applies while the select names none, such as the one that a
   * symbol names. The select then shows it, and offers no "No enumeration".
   */
  inherited?: enumeration.Key;
}

/**
 * Picks an enumeration, or creates one in a modal that starts from the draft, with the
 * search text as its name. A right click on an option edits that enumeration, which
 * changes every use of it.
 */
export const EnumerationSelect = ({
  value,
  onChange,
  draft,
  inherited,
}: EnumerationSelectProps): ReactElement => {
  const canCreate = Access.useCreateGranted(enumeration.TYPE_RESOURCE_ID);
  const prompt = Enumeration.useCreatePrompt();
  const handleCreate = (term: string): void => {
    const start = draft();
    void prompt({ ...start, name: term === "" ? start.name : term }).then((created) => {
      if (created != null) onChange(created);
    });
  };
  const footer = (
    <>
      <Menu.Divider />
      {canCreate && (
        <Select.CreateItem<enumeration.Key, enumeration.Enumeration>
          resourceName="enumeration"
          modal
          onCreate={handleCreate}
        />
      )}
      {inherited == null && (
        <Select.Item<enumeration.Key> itemKey={NONE} textValue="No enumeration">
          <Icon.None />
          No enumeration
        </Select.Item>
      )}
    </>
  );
  return (
    <Enumeration.SelectSingle
      value={value ?? inherited}
      onChange={(next: enumeration.Key | undefined) =>
        onChange(next === NONE ? undefined : next)
      }
      allowNone={inherited == null}
      footer={footer}
    />
  );
};

export interface EnumerationFieldProps {
  /** Path to the enumeration that the node names. */
  path: deep.Path<enumeration.Key | undefined>;
  /** Path to the node's label, which names an enumeration that the node creates. */
  label: deep.Path<string>;
  /** Names an enumeration that the node creates when it has no label. */
  symbolName: string;
}

/** Picks the enumeration that a node draws. */
export const EnumerationField = ({
  path,
  label,
  symbolName,
}: EnumerationFieldProps): ReactElement => {
  const { set, remove, get } = Form.useContext();
  const key = Form.useFieldValue(path, { optional: true }) ?? undefined;
  return (
    <Input.Item label="Enumeration" padHelpText={false}>
      <EnumerationSelect
        value={key}
        onChange={(next) => (next == null ? remove(path) : set(path, next))}
        draft={() => ({
          name: get<string>(label).value.trim() || symbolName,
          entries: Enumeration.createEntries(),
        })}
      />
    </Input.Item>
  );
};
