// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";

import { Confirm } from "@/confirm";

interface UseConfirmDeleteProps {
  type: string;
  description?: string;
}

export const useConfirmDelete = ({
  type,
  description = "This action cannot be undone.",
}: UseConfirmDeleteProps) => {
  const confirm = Confirm.useModal();
  return async (resources: ontology.Resource[]): Promise<boolean> => {
    let message = `Are you sure you want to delete ${resources.length} ${type.toLowerCase()}s?`;
    if (resources.length === 1)
      message = `Are you sure you want to delete ${resources[0].name}?`;
    return await confirm(
      {
        message,
        description,
        confirm: { variant: "error", label: "Delete" },
        cancel: { label: "Cancel" },
      },
      {
        name: `${type}.Delete`,
        icon: type,
      },
    );
  };
};
