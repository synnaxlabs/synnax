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
import { useEffect } from "react";

export const byName = (a: { name: string }, b: { name: string }): number =>
  a.name.localeCompare(b.name);

interface Retrievable {
  secrets: boolean;
  certificates: boolean;
}

/** Returns which of secrets and certificates the user can retrieve. */
export const useRetrievable = (): Retrievable => {
  const secrets = Access.useRetrieveGranted(secret.TYPE_ONTOLOGY_ID);
  const certificates = Access.useRetrieveGranted(certificate.TYPE_ONTOLOGY_ID);
  return { secrets, certificates };
};

export interface Lists {
  /** Null when the user cannot retrieve secrets. */
  secrets: Flux.UseListReturn<Secret.ListQuery, secret.Key, secret.Secret> | null;
  /** Null when the user cannot retrieve certificates. */
  certificates: Flux.UseListReturn<
    Certificate.ListQuery,
    certificate.Key,
    certificate.Certificate
  > | null;
}

/**
 * Retrieves the secrets and certificates the user can read, sorted by name, and keeps
 * them current.
 */
export const useLists = (): Lists => {
  const retrievable = useRetrievable();
  const secrets = Secret.useList({ sort: byName });
  const certificates = Certificate.useList({ sort: byName });
  const { retrieve: retrieveSecrets } = secrets;
  const { retrieve: retrieveCertificates } = certificates;
  useEffect(() => {
    if (retrievable.secrets) retrieveSecrets({});
  }, [retrievable.secrets, retrieveSecrets]);
  useEffect(() => {
    if (retrievable.certificates) retrieveCertificates({});
  }, [retrievable.certificates, retrieveCertificates]);
  return {
    secrets: retrievable.secrets ? secrets : null,
    certificates: retrievable.certificates ? certificates : null,
  };
};
