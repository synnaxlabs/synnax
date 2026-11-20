// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { label } from "@synnaxlabs/client";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import type { ReactElement } from "react";

import { useCreateModal } from "@/platform/label/useCreateModal";

export interface CreateFooterProps {
  /** Called with the key of a label the user creates. */
  onCreate: (key: label.Key) => void;
}

/**
 * The create option of a label select, under a divider. It opens the create modal with
 * the search text as the name, so the user picks the color.
 */
export const CreateFooter = ({ onCreate }: CreateFooterProps): ReactElement => {
  const openCreate = useCreateModal();
  const handleCreate = (name: string) =>
    void openCreate({ name }).then((key) => {
      if (key != null) onCreate(key);
    });
  return (
    <>
      <Menu.Divider />
      <Select.CreateItem<label.Key, label.Label>
        resourceName="label"
        modal
        onCreate={handleCreate}
      />
    </>
  );
};
