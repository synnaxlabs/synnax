// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/media";
import "@synnaxlabs/pluto/dist/pluto.css";
import "@synnaxlabs/media/dist/media.css";

import "./Intro.css";

const TRANSLATION = "translate(-50%, -50%)";

export const MyComposition = () => {
  return (
    <div style={{ width: "100%", height: "100%", backgroundColor: "black" }}>
      <div className="logos">
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
        <Logo style={{ transform: TRANSLATION }} />
      </div>
    </div>
  );
};
