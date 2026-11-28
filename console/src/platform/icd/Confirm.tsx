// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd, task } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Task } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { Modals } from "@/platform/modals";
import { UsedBy } from "@/platform/used-by";

export interface ConfirmParams extends Modals.ConfirmParams {
  /** The ICD that the edit changes. */
  icd: Pick<icd.ICD, "key" | "name">;
}

const Confirm = ({
  icd: { key, name },
  close,
  ...rest
}: Modals.ContentProps<ConfirmParams, boolean>): ReactElement => {
  const users = Task.useResultMultiple({ where: { icds: [key] } });
  const tasks = users.data ?? [];
  return (
    <Modals.ConfirmLayout
      {...rest}
      onConfirm={() => close(true)}
      onCancel={() => close(false)}
    >
      {users.variant === "error" && (
        <Status.Summary level="small" status={users.status} />
      )}
      {tasks.length > 0 && (
        <Flex.Box y gap="small">
          <Text.Text level="small" color={9}>
            Tasks that use {name.trim() === "" ? "this ICD" : name}
          </Text.Text>
          <UsedBy.List
            kind="ICD"
            uses={tasks.map((t) => ({ id: task.resourceID(t.key) }))}
            onSelect={() => close(false)}
          />
        </Flex.Box>
      )}
    </Modals.ConfirmLayout>
  );
};

/**
 * Opens a confirm of an edit to an ICD that lists the tasks that use the ICD. Resolves
 * with true when the user confirms. A click on a task opens it and cancels the edit.
 */
export const useConfirm = Modals.createPrompt<boolean, ConfirmParams>(Confirm);
