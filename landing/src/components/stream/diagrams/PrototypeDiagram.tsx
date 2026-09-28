// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ReactElement } from "react";

import type { CalcDiagramState } from "@/components/stream/calcTimeline";
import {
  ALARM_DIAGRAM,
  CONVERSION_DIAGRAM,
  FFT_DIAGRAM,
  MASSFLOW_DIAGRAM,
  MIXTURE_DIAGRAM,
  VOTING_DIAGRAM,
} from "@/components/stream/diagrams/definitions";
import { Diagram } from "@/components/stream/diagrams/Diagram";
import type { DiagramDef } from "@/components/stream/diagrams/types";

const DIAGRAMS: Record<string, DiagramDef> = {
  conversion: CONVERSION_DIAGRAM,
  mixture: MIXTURE_DIAGRAM,
  alarm: ALARM_DIAGRAM,
  massflow: MASSFLOW_DIAGRAM,
  voting: VOTING_DIAGRAM,
  fft: FFT_DIAGRAM,
};

interface PrototypeDiagramProps {
  diagramKey: string;
  state: CalcDiagramState;
}

export const PrototypeDiagram = ({
  diagramKey,
  state,
}: PrototypeDiagramProps): ReactElement | null => {
  const def = DIAGRAMS[diagramKey];
  if (def == null) return null;
  return <Diagram def={def} state={state} />;
};
