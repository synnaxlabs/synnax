import { type ReactElement, useId } from "react";

import {
  ACCENT,
  ERROR_ACCENT,
  WARNING_ACCENT,
} from "@/components/diagrams/diagramConstants";

interface DiagramDefsIds {
  accentFlow: string;
  accentEndpoint: string;
  errorFlow: string;
  errorEndpoint: string;
  warningFlow: string;
  warningEndpoint: string;
}

export const useDiagramDefs = (): DiagramDefsIds => {
  const id = useId();
  return {
    accentFlow: `${id}-accent-flow`,
    accentEndpoint: `${id}-accent-ep`,
    errorFlow: `${id}-error-flow`,
    errorEndpoint: `${id}-error-ep`,
    warningFlow: `${id}-warning-flow`,
    warningEndpoint: `${id}-warning-ep`,
  };
};

const FlowGradient = ({
  id,
  color,
}: {
  id: string;
  color: string;
}): ReactElement => (
  <linearGradient id={id} gradientUnits="userSpaceOnUse">
    <stop offset="0%" stopColor={color} stopOpacity="0">
      <animate
        attributeName="stopOpacity"
        values="0;0.6;0"
        dur="2.5s"
        repeatCount="indefinite"
      />
    </stop>
    <stop offset="40%" stopColor={color} stopOpacity="0.5">
      <animate
        attributeName="stopOpacity"
        values="0.5;0.8;0.5"
        dur="2.5s"
        repeatCount="indefinite"
      />
    </stop>
    <stop offset="100%" stopColor={color} stopOpacity="0">
      <animate
        attributeName="stopOpacity"
        values="0;0.4;0"
        dur="2.5s"
        repeatCount="indefinite"
      />
    </stop>
  </linearGradient>
);

const EndpointGlow = ({
  id,
  color,
}: {
  id: string;
  color: string;
}): ReactElement => (
  <radialGradient id={id} cx="50%" cy="50%" r="50%">
    <stop offset="0%" stopColor={color} stopOpacity="0.15" />
    <stop offset="100%" stopColor={color} stopOpacity="0" />
  </radialGradient>
);

export const DiagramDefs = ({
  ids,
}: {
  ids: DiagramDefsIds;
}): ReactElement => (
  <defs>
    <FlowGradient id={ids.accentFlow} color={ACCENT} />
    <EndpointGlow id={ids.accentEndpoint} color={ACCENT} />
    <FlowGradient id={ids.errorFlow} color={ERROR_ACCENT} />
    <EndpointGlow id={ids.errorEndpoint} color={ERROR_ACCENT} />
    <FlowGradient id={ids.warningFlow} color={WARNING_ACCENT} />
    <EndpointGlow id={ids.warningEndpoint} color={WARNING_ACCENT} />
  </defs>
);
