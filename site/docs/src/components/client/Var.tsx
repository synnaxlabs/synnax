// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { VAR_ATTRIBUTE } from "@/components/client/choice";
import { QUERY_ATTRIBUTE } from "@/components/tabs/sync";

export interface VarProps {
  py: string;
  ts: string;
}

/** Shows an identifier in the naming style of the chosen client. */
export const Var = ({ py, ts }: VarProps): ReactElement => (
  <code
    {...{ [QUERY_ATTRIBUTE]: "client", [VAR_ATTRIBUTE]: "" }}
    data-py={py}
    data-ts={ts}
  >
    {py}
  </code>
);
