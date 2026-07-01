// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, UnexpectedError } from "@synnaxlabs/client";
import { Arc, type Flux, type List } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Modals } from "@/primitive/modals";
import { Session } from "@/session";

export const useRename = (
  getItem: List.GetItem<arc.Key, arc.Arc>,
): { update: (params: Arc.RenameParams) => void } => {
  const dispatch = Session.useDispatch();
  const confirm = Modals.useConfirm();
  const { update } = Arc.useRename({
    beforeUpdate: useCallback(
      async ({
        data,
        rollbacks,
        store,
        client,
      }: Flux.BeforeUpdateParams<Arc.RenameParams, false, Arc.FluxSubStore>) => {
        const { key, name } = data;
        const tsk = await Arc.retrieveTask({
          store,
          client,
          query: { arcKey: key },
        });
        const a = getItem(key);
        if (a == null) throw new UnexpectedError(`Arc with key ${key} not found`);
        const oldName = a.name;
        if (tsk?.status?.details.running === true) {
          const confirmed = await confirm({
            message: `Are you sure you want to rename ${a.name} to ${name}?`,
            description: `This will cause ${a.name} to stop and be reconfigured.`,
            cancel: { label: "Cancel" },
            confirm: { label: "Rename", variant: "error" },
          });
          if (!confirmed) return false;
        }
        dispatch(Session.Layout.rename({ key, name }));
        rollbacks.push(() => dispatch(Session.Layout.rename({ key, name: oldName })));
        return data;
      },
      [dispatch, getItem, confirm],
    ),
  });
  return { update };
};
