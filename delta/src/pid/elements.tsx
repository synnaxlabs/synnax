// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { PIDElement } from "@synnaxlabs/pluto";

export const ELEMENTS: Record<string, PIDElement.Spec<any>> = {
  [PIDElement.ValueSpec.type]: PIDElement.ValueSpec,
  [PIDElement.RegulatorSpec.type]: PIDElement.RegulatorSpec,
  [PIDElement.ValveSpec.type]: PIDElement.ValveSpec,
  [PIDElement.TankSpec.type]: PIDElement.TankSpec,
};
