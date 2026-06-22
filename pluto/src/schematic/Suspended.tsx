// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type schematic } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";

import { useEnsureRetrieved } from "./queries";

const [KeyContext, useOptionalKey] = context.create<schematic.Key | undefined>({
  displayName: "KeyContext.Provider",
  defaultValue: undefined,
});

export const useKey = (override?: schematic.Key): schematic.Key => {
  const value = useOptionalKey() ?? override;
  if (value == null)
    throw new NotFoundError(
      `Schematic.useKey must be passed an override or called inside of Schematic.KeyContext`,
    );
  return value;
};

export interface SuspendedProps extends PropsWithChildren {
  schematicKey: schematic.Key;
}

export const Suspended = ({ schematicKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: schematicKey });
  return <KeyContext value={schematicKey}>{children}</KeyContext>;
};
