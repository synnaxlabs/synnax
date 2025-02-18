// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, type Name, ONTOLOGY_TYPE } from "@/ontology/group/payload";
import * as ontology from "@/ontology/payload";

export class Group {
  key: Key;
  name: Name;

  constructor(name: Name, key: Key) {
    this.key = key;
    this.name = name;
  }

  get ontologyID(): ontology.ID {
    return ontologyID(this.key);
  }
}

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
