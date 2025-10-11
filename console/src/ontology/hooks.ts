// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/client";
import { array, strings } from "@synnaxlabs/x";

import { Modals } from "@/modals";

interface UseConfirmDeleteProps {
  type: string;
  icon?: string;
  description?: string;
}

const CONFIRM_DELETE_DELAY = TimeSpan.milliseconds(0);

interface ConfirmDeleteItem {
  name: string;
}
export const useConfirmDelete = ({
  type,
  icon,
  description = "This action cannot be undone.",
}: UseConfirmDeleteProps) => {
  const confirm = Modals.useConfirm();
  return async (items_: ConfirmDeleteItem | ConfirmDeleteItem[]): Promise<boolean> => {
    const items = array.toArray(items_);
    let message = `Are you sure you want to delete ${items.length} ${strings.pluralName(type.toLowerCase())}?`;
    if (items.length === 1)
      message = `Are you sure you want to delete ${items[0].name}?`;
    return (
      (await confirm(
        {
          message,
          description,
          confirm: {
            variant: "error",
            label: "Delete",
            delay: CONFIRM_DELETE_DELAY.milliseconds,
          },
          cancel: { label: "Cancel" },
        },
        { name: `${type}.Delete`, icon: icon ?? type },
      )) ?? false
    );
  };
};
