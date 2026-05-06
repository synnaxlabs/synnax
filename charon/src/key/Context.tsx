// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren } from "react";

import { context } from "@/context";

export const [BaseProvider, baseUse] = context.create<record.Key>({
  displayName: "Key.Context",
  providerName: "Key.Provider",
});

export const use = <K extends record.Key>(hookOrComponentName: string): K =>
  baseUse(hookOrComponentName) as K;

export interface ProviderProps<K extends record.Key> extends PropsWithChildren {
  value: K;
}

export const Provider = <K extends record.Key>(props: ProviderProps<K>) => (
  <BaseProvider {...props} />
);
