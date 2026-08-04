// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { plural } from "pluralize";

import { useConfirm } from "@/platform/modals/useConfirm";

interface UseConfirmDeleteProps {
  /** Singular, capitalized noun for the resource, e.g. "Range". */
  type: string;
  /** Dotted breadcrumb path for the header. Defaults to `${type}.Delete`. */
  title?: string;
  icon?: string;
  description?: string;
}

interface ConfirmDeleteItem {
  name: string;
}

/**
 * Prompts for confirmation before deleting resources. One item is named; two or more
 * collapse to a count, keeping the message bounded however large the selection.
 */
export const useConfirmDelete = ({
  type,
  title = `${type}.Delete`,
  icon,
  description = "This action cannot be undone.",
}: UseConfirmDeleteProps) => {
  const confirm = useConfirm();
  return async (items_: ConfirmDeleteItem | ConfirmDeleteItem[]): Promise<boolean> => {
    const items = array.toArray(items_);
    let message = `Are you sure you want to delete ${items.length} ${plural(type.toLowerCase())}?`;
    if (items.length === 1)
      message = `Are you sure you want to delete ${items[0].name}?`;
    return (
      (await confirm({
        message,
        description,
        confirm: { variant: "error", label: "Delete" },
        cancel: { label: "Cancel" },
        title,
        icon: Icon.resolve(icon ?? type),
      })) ?? false
    );
  };
};
