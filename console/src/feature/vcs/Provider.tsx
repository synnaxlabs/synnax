// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { vcs } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { useDiscard } from "@/feature/vcs/actions";
import { Diff } from "@/feature/vcs/diff";
import { Vcs } from "@/platform/vcs";

const DiscardAction = ({ target }: Diff.ActionsProps): ReactElement | null => {
  const discard = useDiscard();
  const canUpdate = Access.useUpdateGranted(Vcs.RESOURCE_ID);
  if (!canUpdate || target.before !== vcs.HEAD || target.after !== vcs.WORKING)
    return null;
  const { resource, name } = target;
  return (
    <Button.Button
      variant="outlined"
      size="small"
      onClick={() => discard([{ resource, name }])}
    >
      <Icon.Discard />
      Discard changes
    </Button.Button>
  );
};

/** Holds the version control status, the toolbar state, and the diff tab target. */
export const Provider = ({ children }: PropsWithChildren): ReactElement => (
  <Vcs.Provider>
    <Diff.Provider Actions={DiscardAction}>{children}</Diff.Provider>
  </Vcs.Provider>
);
