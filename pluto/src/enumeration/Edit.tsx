// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/enumeration/Edit.css";

import { enumeration, schematic } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import type { ReactElement } from "react";

import { Access } from "@/access";
import { CSS } from "@/css";
import { Entries } from "@/enumeration/Entries";
import { useRename, useResult } from "@/enumeration/queries";
import { Synnax } from "@/synnax";

const note = (uses: number): string =>
  uses < 2
    ? "Changes every use of this enumeration."
    : `Changes every use of this enumeration, in ${uses} schematics.`;

interface EditViewProps {
  itemKey: enumeration.Key;
}

const EditView = ({ itemKey }: EditViewProps): ReactElement => {
  const { data } = useResult({ key: itemKey, includeUses: true });
  const { update: rename } = useRename();
  const uses = (data?.uses ?? []).filter(
    ({ type }) => type === schematic.TYPE_RESOURCE_ID.type,
  ).length;
  return (
    <>
      <Text.Text level="small" color={9} className={CSS.BE("enumeration-edit", "note")}>
        {note(uses)}
      </Text.Text>
      <Input.Text
        value={data?.name ?? ""}
        onChange={(name) => {
          const trimmed = name.trim();
          if (trimmed !== "") rename({ key: itemKey, name: trimmed });
        }}
        onlyChangeOnBlur
        placeholder="Enumeration name"
        aria-label="Enumeration name"
        size="small"
        className={CSS.BE("enumeration-edit", "name")}
      />
      <Entries enumeration={itemKey} />
    </>
  );
};

/**
 * The context menu of an option in an enumeration select. Its Edit item opens the
 * editor of the enumeration in place of the options. Each edit saves at once.
 */
export const OptionMenu = ({
  keys: [key],
}: Menu.ContextMenuMenuProps): ReactElement => {
  const { open } = Dialog.useViews();
  return (
    <Menu.Menu gap="small">
      <Menu.Item
        itemKey="edit"
        onClick={() => open("Edit enumeration", <EditView itemKey={key} />)}
      >
        <Icon.Edit />
        Edit
      </Menu.Item>
    </Menu.Menu>
  );
};

/**
 * @returns whether the option with the key opens {@link OptionMenu}: only when the user
 * can update the enumeration.
 */
export const useHasOptionMenu = (): ((key: enumeration.Key) => boolean) => {
  const client = Synnax.use();
  return (key) => Access.updateGranted({ client, id: enumeration.resourceID(key) });
};
