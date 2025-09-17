// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import {
  groupZ,
  type Key,
  keyZ,
  type Name,
  nameZ,
  type Group,
} from "@/ontology/group/payload";
import { type ID as OntologyID, idZ as ontologyIDZ } from "@/ontology/payload";

export class Writer {}
