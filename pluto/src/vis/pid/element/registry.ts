// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ButtonSpec } from "@/vis/pid/element/Button";
import { type Spec } from "@/vis/pid/element/element";
import { RegulatorSpec } from "@/vis/pid/element/Regulator";
import { TankSpec } from "@/vis/pid/element/Tank";
import { ValueSpec } from "@/vis/pid/element/Value";
import { ValveSpec } from "@/vis/pid/element/Valve";

export const REGISTRY: Record<string, Spec<any>> = {
  [ValveSpec.type]: ValveSpec,
  [TankSpec.type]: TankSpec,
  [RegulatorSpec.type]: RegulatorSpec,
  [ValueSpec.type]: ValueSpec,
  [ButtonSpec.type]: ButtonSpec,
};
