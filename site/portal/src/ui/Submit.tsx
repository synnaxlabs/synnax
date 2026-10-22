// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { type PropsWithChildren, type ReactElement } from "react";

import { type Action } from "@/ui/useAction";

export interface SubmitProps extends PropsWithChildren {
  action: Action;
}

/** Submit is a form's primary button. It runs `action` on click or Ctrl+Enter. */
export const Submit = ({ action, children }: SubmitProps): ReactElement => (
  <Button.Button
    variant="filled"
    onClick={action.run}
    status={action.loading ? "loading" : undefined}
    trigger={["Control", "Enter"]}
    triggerIndicator
  >
    {children}
  </Button.Button>
);
