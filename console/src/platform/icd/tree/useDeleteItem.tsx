// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { unique } from "@synnaxlabs/x";
import type { ReactNode } from "react";

import { useConfirm } from "@/platform/icd/Confirm";
import { formatOf, heldMessage, useFormats } from "@/platform/icd/Formats";

/** A field to delete from the ICD, at its message. */
export interface DeletedField {
  icd: icd.Key;
  message: icd.MessageKey;
  field: { key: string; name: string };
}

const blank = (text: string, fallback: string): string =>
  text.trim() === "" ? fallback : text;

/**
 * Returns a function that makes the "Delete from ICD" item of a menu on fields. A click
 * asks first, and calls onDelete once the user confirms. Fields of several ICDs get no
 * item. Must be inside ICD.WithFormats.
 */
export const useDeleteItem = (): ((
  fields: DeletedField[],
  onDelete: () => void,
) => ReactNode) => {
  const formats = useFormats();
  const confirm = useConfirm();
  const ask = async (key: icd.Key, fields: DeletedField[], onDelete: () => void) => {
    const names = new Map(fields.map((f) => [`${f.message}/${f.field.key}`, f.field]));
    const messages = unique.unique(fields.map(({ message }) => message));
    const { name } = formatOf(formats, key).value();
    const held =
      messages.length === 1 ? heldMessage(formats, key, messages[0]) : undefined;
    const from =
      held == null ? blank(name, "this ICD") : blank(held.name, "this message");
    const [first] = names.values();
    const one = names.size === 1;
    const what = one ? blank(first.name, "this field") : `${names.size} fields`;
    const [it, those] = one ? ["it", "that channel"] : ["them", "those channels"];
    const theICD = name.trim() === "" ? "this ICD" : `the ICD ${name}`;
    const loss = `Tasks that bind ${it} lose ${those}.`;
    const confirmed = await confirm({
      icd: { key, name },
      title: one ? "ICD.Delete field" : "ICD.Delete fields",
      message: `Delete ${what} from ${from}?`,
      description: `This deletes ${it} from ${theICD}. ${loss}`,
      confirm: { label: "Delete" },
    });
    if (confirmed === true) onDelete();
  };
  return (fields, onDelete) => {
    const icds = unique.unique(fields.map(({ icd }) => icd));
    if (icds.length !== 1) return null;
    return (
      <Menu.Item itemKey="delete" onClick={() => void ask(icds[0], fields, onDelete)}>
        <Icon.Delete />
        Delete from ICD
      </Menu.Item>
    );
  };
};
