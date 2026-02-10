// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { type Services } from "@/ontology/service";

const [Context, useContext] = context.create<Services>({
  displayName: "Ontology.Context",
  providerName: "Ontology.ServicesProvider",
});

export const useServices = (): Services => useContext("Ontology.useServices");

export const useService = <T extends keyof Services>(service: T): Services[T] => {
  const services = useContext("Ontology.useService");
  return services[service];
};

export interface ServicesProviderProps extends PropsWithChildren {
  services: Services;
}

export const ServicesProvider = ({
  services,
  ...rest
}: ServicesProviderProps): ReactElement => <Context value={services} {...rest} />;
