// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Alamos, Pluto } from "@synnaxlabs/pluto";
import { memo } from "react";

import { Triggers } from "@/app/triggers";
import { Embedded } from "@/feature/embedded";
import { Session } from "@/session";

export interface ContextProps extends Pick<
  Pluto.ProviderProps,
  "workerURL" | "children"
> {}

const ALAMOS_PROPS: Alamos.ProviderProps = { level: "info" };

const useConnParams = DESKTOP ? Embedded.useConnParams : Session.Core.useSelectSelected;

export const Context = memo((props: ContextProps) => {
  const connParams = useConnParams();
  const themingProps = Session.Theme.useProviderProps();
  return (
    <Pluto.Provider
      workerEnabled
      connParams={connParams}
      triggers={Triggers.PROVIDER_PROPS}
      haul={Session.Haul.PROVIDER_PROPS}
      color={Session.Color.PROVIDER_PROPS}
      alamos={ALAMOS_PROPS}
      theming={themingProps}
      {...props}
    />
  );
});
Context.displayName = "Pluto.Context";
