// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax as Client } from "@synnaxlabs/client";
import { context, type Icon } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { type Session } from "@/session";

export interface SnapshotContext {
  client: Client | null;
  placeLayout: Session.Layout.Placer;
}

export interface SnapshotService {
  icon: Icon.ReactElement;
  onClick: (res: ontology.Resource, ctx: SnapshotContext) => Promise<void>;
  onDelete: (res: ontology.Resource, ctx: SnapshotContext) => Promise<void>;
}

export type SnapshotServices = Record<string, SnapshotService>;

const [Context, useContext] = context.create<SnapshotServices>({
  displayName: "Range.SnapshotContext",
  providerName: "Range.SnapshotServicesProvider",
});

export const useSnapshotServices = (): SnapshotServices =>
  useContext("Range.useSnapshotServices");

export interface SnapshotServicesProviderProps extends PropsWithChildren {
  services: SnapshotServices;
}

export const SnapshotServicesProvider = ({
  services,
  ...rest
}: SnapshotServicesProviderProps): ReactElement => <Context value={services} {...rest} />;
