// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type label } from "@synnaxlabs/client";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Synnax } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useCreateModal } from "@/platform/label/useCreateModal";

export interface CreateFooterProps {
  /** Called with the key of a label the user creates. */
  onCreate: (key: label.Key) => void;
}

/**
 * The create option of a label select, under a divider. A new label takes the first
 * palette color that no label uses.
 */
export const CreateFooter = ({ onCreate }: CreateFooterProps): ReactElement => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const palette = Theming.use().colors.visualization.palettes.default;
  const openCreate = useCreateModal();
  const handleCreate = (name: string) =>
    handleError(
      async () => {
        if (client == null) throw new DisconnectedError();
        const used = new Set(
          (await client.labels.retrieve({})).map((l) => color.hex(l.color)),
        );
        const next = palette.find((c) => !used.has(color.hex(c))) ?? palette[0];
        if (name !== "") {
          const created = await client.labels.create({ name, color: next });
          return onCreate(created.key);
        }
        const key = await openCreate({ color: next });
        if (key != null) onCreate(key);
      },
      { message: "Failed to create label", name },
    );
  return (
    <>
      <Menu.Divider />
      <Select.CreateItem<label.Key, label.Label>
        resourceName="label"
        onCreate={handleCreate}
      />
    </>
  );
};
