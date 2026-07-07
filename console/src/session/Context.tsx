// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Provider } from "@synnaxlabs/drift/react";
import { useInitializerRef } from "@synnaxlabs/pluto";
import { type ReactNode } from "react";

import { Modals } from "@/session/modals";
import { configureStore } from "@/session/store";

export interface ContextProps {
  children: ReactNode;
}

export const Context = (props: ContextProps) => {
  const storeRef = useInitializerRef(() => configureStore());
  return (
    <Modals.Context>
      <Provider store={storeRef.current} {...props} />
    </Modals.Context>
  );
};
