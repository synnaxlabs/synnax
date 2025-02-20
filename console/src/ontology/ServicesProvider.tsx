// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, type PropsWithChildren, use } from "react";

import { type Services } from "@/ontology/service";

export interface ServicesContextValue extends Services {}

const Context = createContext<ServicesContextValue | null>(null);

export const useServices = () => {
  const services = use(Context);
  if (services == null)
    throw new Error("useServices must be used within a ServicesProvider");
  return services;
};

export interface ServicesProviderProps extends PropsWithChildren {
  services: Services;
}

export const ServicesProvider = ({ services, children }: ServicesProviderProps) => (
  <Context value={services}>{children}</Context>
);
