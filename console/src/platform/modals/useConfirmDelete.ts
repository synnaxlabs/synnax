// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { array } from "@synnaxlabs/x";
import pluralize from "pluralize";

import {
  type ConfirmOption,
  type ConfirmParams,
  useConfirm,
  useConfirmWithOption,
} from "@/platform/modals/useConfirm";

interface UseConfirmDeleteProps {
  /** Singular, capitalized noun for the resource, e.g. "Range". */
  type: string;
  /** Noun for a count of items, e.g. "ICDs". Defaults to the plural of `type`. */
  plural?: string;
  /** Dotted breadcrumb path for the header. Defaults to `${type}.Delete`. */
  title?: string;
  icon?: string;
  description?: string;
}

interface ConfirmDeleteItem {
  name: string;
}

// Calls, since the compiler cannot reorder most expressions in a parameter default.
const deleteTitle = (type: string): string => `${type}.Delete`;
const deletePlural = (type: string): string => pluralize.plural(type.toLowerCase());

type ConfirmDeleteItems = ConfirmDeleteItem | ConfirmDeleteItem[];

// The React Compiler leaves an overloaded function declaration in a hook unset on the
// first render, so the overloads live outside the hook.
const createConfirmDelete = (
  confirm: ReturnType<typeof useConfirm>,
  confirmWithOption: ReturnType<typeof useConfirmWithOption>,
  paramsOf: (items: ConfirmDeleteItems) => ConfirmParams,
) => {
  function confirmDelete(
    items: ConfirmDeleteItems,
    option: ConfirmOption,
  ): Promise<{ option: boolean } | null>;
  function confirmDelete(items: ConfirmDeleteItems): Promise<boolean>;
  async function confirmDelete(items: ConfirmDeleteItems, option?: ConfirmOption) {
    const params = paramsOf(items);
    if (option == null) return (await confirm(params)) ?? false;
    return await confirmWithOption({ ...params, option });
  }
  return confirmDelete;
};

/**
 * Prompts for confirmation before deleting resources. One item is named; two or more
 * collapse to a count, keeping the message bounded however large the selection. With an
 * option, the prompt shows it as a checkbox and resolves with its state, or null when
 * the user cancels.
 */
export const useConfirmDelete = ({
  type,
  plural = deletePlural(type),
  title = deleteTitle(type),
  icon,
  description = "This action cannot be undone.",
}: UseConfirmDeleteProps) => {
  const confirm = useConfirm();
  const confirmWithOption = useConfirmWithOption();
  const paramsOf = (items_: ConfirmDeleteItems): ConfirmParams => {
    const items = array.toArray(items_);
    let message = `Are you sure you want to delete ${items.length} ${plural}?`;
    if (items.length === 1)
      message = `Are you sure you want to delete ${items[0].name}?`;
    return {
      message,
      description,
      confirm: { variant: "error", label: "Delete" },
      cancel: { label: "Cancel" },
      title,
      icon: Icon.resolve(icon ?? type),
    };
  };
  return createConfirmDelete(confirm, confirmWithOption, paramsOf);
};
