// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";

import { Sync } from "@/sync";

const parseID = (str: string): ontology.ID => new ontology.ID(str);

export const useResourceSetSynchronizer = (onSet: (id: ontology.ID) => void): void =>
  Sync.useStringListener(ontology.RESOURCE_SET_CHANNEL_NAME, parseID, onSet);

export const useResourceDeleteSynchronizer = (
  onDelete: (id: ontology.ID) => void,
): void =>
  Sync.useStringListener(ontology.RESOURCE_DELETE_CHANNEL_NAME, parseID, onDelete);
