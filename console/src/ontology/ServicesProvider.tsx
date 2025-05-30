// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, type PropsWithChildren, type ReactElement, use } from "react";

import { type Services } from "@/ontology/service";

const Context = createContext<Services | null>(null);

export const useServices = (): Services => {
  const services = use(Context);
  if (services == null)
    throw new Error("useServices must be used within a ServicesProvider");
  return services;
};

export interface ServicesProviderProps extends PropsWithChildren {
  services: Services;
}

export const ServicesProvider = ({
  services,
  ...rest
}: ServicesProviderProps): ReactElement => <Context value={services} {...rest} />;
