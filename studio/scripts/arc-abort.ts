// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import {
  button,
  controls,
  editor,
  openAutomation,
  openSample,
  symbol,
  tab,
  watchTank,
} from "./common/sample";

export const desktop = true;

const HOLD_AND_VENT = `import time

sequence press {
    stage idle {
        0 -> sample_press_vlv_cmd
        0 -> sample_vent_vlv_cmd
        sample_start_cmd == 1 => pressurize
    }
    stage pressurize {
        1 -> sample_press_vlv_cmd
        0 -> sample_vent_vlv_cmd
        sample_tank_pt > 300 => hold
    }
    stage hold {
        0 -> sample_press_vlv_cmd
        time.wait{duration=10s} => vent
    }
    stage vent {
        1 -> sample_vent_vlv_cmd
        sample_tank_pt < 5 => idle
    }
}

true => press
`;

const ABORT = `import time

sequence press {
    stage idle {
        0 -> sample_press_vlv_cmd
        0 -> sample_vent_vlv_cmd
        sample_start_cmd == 1 => pressurize
    }
    stage pressurize {
        1 -> sample_press_vlv_cmd
        0 -> sample_vent_vlv_cmd
        sample_tank_pt > 450 => abort
        sample_abort_cmd == 1 => abort
        sample_tank_pt > 300 => hold
    }
    stage hold {
        0 -> sample_press_vlv_cmd
        sample_abort_cmd == 1 => abort
        time.wait{duration=10s} => vent
    }
    stage vent {
        1 -> sample_vent_vlv_cmd
        sample_tank_pt < 5 => idle
    }
    stage abort {
        0 -> sample_press_vlv_cmd
        1 -> sample_vent_vlv_cmd
        sample_start_cmd == 1 => pressurize
    }
}

true => press
`;

/**
 * Docs `control/arc/get-started/abort`: replace the hold and vent version of the
 * sequence with the abort version and redeploy it. Click Start on the Tank schematic,
 * then Abort while the pressure climbs: the press valve closes, the vent valve opens,
 * and the pressure falls.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await openSample(session);
  await openAutomation(session, {
    name: "My press sequence",
    text: HOLD_AND_VENT,
    running: true,
  });
  const tank = await watchTank();
  try {
    const redeploy = controls(page).getByRole("button", { name: "Redeploy" });
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);
    await session.zoom(editor(page), 1.4);
    await session.click(editor(page));
    await session.press("ControlOrMeta+a");
    await session.hold(300);
    await session.paste(ABORT);
    await session.hold(1200);
    await session.zoom(controls(page), 1.6);
    await session.waitFor(redeploy);
    await session.click(redeploy);
    await session.waitForHidden(redeploy);
    await session.hold(1000);
    session.endZoom();
    await session.click(tab(page, "Tank"), { zoom: false });
    await session.waitFor(symbol(page, "Start"));
    await session.hold(600);
    await session.zoom([symbol(page, "Start"), symbol(page, "Abort")], 1.6);
    await session.click(button(page, "Start"));
    await session.waitFor(() => tank.press === 1);
    // After the hold, the sequence vents by itself, and the shot would show no abort.
    if (tank.vent === 1) throw new Error("the test vented before the Abort click");
    await session.click(button(page, "Abort"));
    session.endZoom();
    await session.moveTo(capture.PARK);
    await session.waitFor(() => tank.press === 0 && tank.vent === 1);
    await session.waitFor(() => tank.pressure < 20, 3600);
    await session.hold(2500);
  } finally {
    await tank.stop();
  }
};
