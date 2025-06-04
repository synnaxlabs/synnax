// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { useState } from "react";

import { NULL_CLIENT_ERROR } from "@/errors";
import { useAsyncEffect } from "@/hooks";
import {
  useResourceDeleteSynchronizer,
  useResourceSetSynchronizer,
} from "@/ontology/synchronizers";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

export const useResource = (id: ontology.ID): ontology.Resource | null => {
  const client = Synnax.use();
  const [resource, setResource] = useState<ontology.Resource | null>(null);
  useAsyncEffect(async () => {
    if (client == null) throw NULL_CLIENT_ERROR;
    const resource = await client.ontology.retrieve(id);
    setResource(resource);
  }, [client, id]);
  const handleError = Status.useErrorHandler();
  const handleResourceSet = (id: ontology.ID) => {
    if (!id.equals(id)) return;
    handleError(async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const resource = await client.ontology.retrieve(id);
      setResource(resource);
    }, "Failed to retrieve resource");
  };
  useResourceSetSynchronizer(handleResourceSet);
  const handleResourceDelete = (id: ontology.ID) => {
    if (!id.equals(id)) return;
    setResource(null);
  };
  useResourceDeleteSynchronizer(handleResourceDelete);
  return resource;
};
