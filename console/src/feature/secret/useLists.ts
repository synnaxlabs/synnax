// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, type secret } from "@synnaxlabs/client";
import { Certificate, type Flux, Secret } from "@synnaxlabs/pluto";
import { useEffect } from "react";

const byName = (a: { name: string }, b: { name: string }): number =>
  a.name.localeCompare(b.name);

export interface Lists {
  secrets: Flux.UseListReturn<Secret.ListQuery, secret.Key, secret.Secret>;
  certificates: Flux.UseListReturn<
    Certificate.ListQuery,
    certificate.Key,
    certificate.Certificate
  >;
}

/** Retrieves every secret and certificate, sorted by name, and keeps them current. */
export const useLists = (): Lists => {
  const secrets = Secret.useList({ sort: byName });
  const certificates = Certificate.useList({ sort: byName });
  const { retrieve: retrieveSecrets } = secrets;
  const { retrieve: retrieveCertificates } = certificates;
  useEffect(() => {
    retrieveSecrets({});
    retrieveCertificates({});
  }, [retrieveSecrets, retrieveCertificates]);
  return { secrets, certificates };
};
