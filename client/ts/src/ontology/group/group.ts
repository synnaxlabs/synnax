// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@/ontology";
import { ontologyID as payloadOntologyID } from "@/ontology/group/payload";

export class Group {
  key: string;
  name: string;

  constructor(name: string, key: string) {
    this.key = key;
    this.name = name;
  }

  get ontologyID(): ontology.ID {
    return payloadOntologyID(this.key);
  }
}
