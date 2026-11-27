// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { ICD, Task } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { Modals } from "@/platform/modals";
import { UsedBy } from "@/platform/used-by";

const CANNOT_UNDO = "This action cannot be undone.";

interface DeleteParams {
  icds: Pick<icd.ICD, "key" | "name">[];
}

const Delete = ({
  icds,
  close,
}: Modals.ContentProps<DeleteParams, void>): ReactElement => {
  const users = Task.useResultMultiple({ where: { icds: icds.map(({ key }) => key) } });
  const { update: remove, variant } = ICD.useDelete({ afterSuccess: () => close() });
  const busy = variant === "loading";
  Modals.useBusy(busy);
  const tasks = users.data ?? [];
  const inUse = tasks.length > 0;
  const single = icds.length === 1 ? icds[0] : undefined;
  const name = single?.name ?? `${icds.length} ICDs`;
  let message = `Are you sure you want to delete ${name}?`;
  let description = CANNOT_UNDO;
  if (inUse) {
    message = single == null ? "Some of these ICDs are in use" : `${name} is in use`;
    description = `Remove ${single == null ? "them" : "it"} from these tasks first.`;
  }
  return (
    <Modals.ConfirmLayout
      title="ICD.Delete"
      icon={<Icon.Library />}
      message={message}
      description={description}
      confirm={{
        label: "Delete",
        variant: busy ? "loading" : "error",
        disabled: users.variant !== "success" || inUse,
      }}
      cancel={{ disabled: busy }}
      onConfirm={() => remove(icds.map(({ key }) => key))}
      onCancel={() => close()}
    >
      {users.variant === "error" && (
        <Status.Summary level="small" status={users.status} />
      )}
      {inUse && (
        <UsedBy.List
          kind="ICD"
          uses={tasks.map(({ key }) => ({ id: task.resourceID(key) }))}
        />
      )}
    </Modals.ConfirmLayout>
  );
};

/**
 * Opens a dialog that deletes ICDs. While a task uses one of them, the dialog names the
 * tasks and offers no Delete.
 */
export const useDeleteModal = Modals.create(Delete);
