// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, secret } from "@synnaxlabs/client";
import { Access, Certificate, type Flux, Secret } from "@synnaxlabs/pluto";

interface Retrievable {
  secrets: boolean;
  certificates: boolean;
}

/** Returns which of secrets and certificates the user can retrieve. */
export const useRetrievable = (): Retrievable => {
  const secrets = Access.useRetrieveGranted(secret.TYPE_RESOURCE_ID);
  const certificates = Access.useRetrieveGranted(certificate.TYPE_RESOURCE_ID);
  return { secrets, certificates };
};

export interface Lists {
  /** Null when the user cannot retrieve secrets. */
  secrets: Flux.UseListReturn<secret.Key, secret.Secret> | null;
  /** Null when the user cannot retrieve certificates. */
  certificates: Flux.UseListReturn<certificate.Key, certificate.Certificate> | null;
}

/**
 * Lists the secrets and certificates the user can read, by name, and keeps them
 * current. A list reads nothing until its first call to `more`.
 */
export const useLists = (): Lists => {
  const retrievable = useRetrievable();
  const secrets = Secret.useList();
  const certificates = Certificate.useList();
  return {
    secrets: retrievable.secrets ? secrets : null,
    certificates: retrievable.certificates ? certificates : null,
  };
};
