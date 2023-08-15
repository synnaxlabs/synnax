// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { BooleanTelem } from "@/telem/bool/main";
import { ControlTelem } from "@/telem/control/hooks";
import { RemoteTelem } from "@/telem/remote/main";
import { StaticTelem } from "@/telem/static/main";
import { TelemProvider } from "@/telem/TelemProvider/TelemProvider";

export const Telem = {
  Static: StaticTelem,
  Remote: RemoteTelem,
  Boolean: BooleanTelem,
  Provider: TelemProvider,
  Control: ControlTelem,
};
