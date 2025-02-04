// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, type PropsWithChildren, useContext } from "react";

import { type Services } from "@/ontology/service";

export interface ServicesContextValue extends Services {}

export const ServicesContext = createContext<ServicesContextValue | null>(null);

export const useServices = (): ServicesContextValue => {
  const services = useContext(ServicesContext);
  if (services == null)
    throw new Error("useServices must be used within a ServicesProvider");
  return services;
};

export interface ServicesProviderProps extends PropsWithChildren {
  services: Services;
}

export const ServicesProvider = ({
  services,
  children,
}: ServicesProviderProps): React.ReactElement => (
  <ServicesContext.Provider value={services}>{children}</ServicesContext.Provider>
);
