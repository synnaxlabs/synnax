export type NodeState = "active" | "excluded" | "alarm" | "inactive";

export interface NodeColors {
  tabFill: string;
  tabCenter: string;
  tabBorder: string;
  iconColor: string;
  iconBg: string;
  iconBorder: string;
  textColor: string;
  valueColor: string;
}

const BLUE_BRIGHT = "var(--pluto-primary-p1)";
const BLUE_DARK = "var(--pluto-primary-z-20)";
const RED_BRIGHT = "var(--pluto-error-z)";
const RED_DARK = "var(--pluto-error-z-15)";
const DIM_BORDER = "var(--pluto-gray-l5)";
const DIM_FILL = "var(--pluto-gray-l1)";
const TEXT_BRIGHT = "var(--pluto-gray-l9)";
const TEXT_DIM = "var(--pluto-gray-l5)";

export const NODE_BG = "var(--pluto-gray-l0)";
export const NODE_BORDER = DIM_BORDER;
export const NODE_RX = 6;

export const BLUE_TAB_COLORS: Pick<NodeColors, "tabFill" | "tabCenter" | "tabBorder"> = {
  tabFill: BLUE_DARK,
  tabCenter: BLUE_BRIGHT,
  tabBorder: BLUE_BRIGHT,
};

export const PARTICLE_COLOR = BLUE_BRIGHT;
export const PARTICLE_COUNT = 4;

const BLUE_TRACE = "var(--pluto-primary-z-15)";
const DIM_TRACE = "var(--pluto-gray-l2)";

export const resolveNodeColors = (state: NodeState): NodeColors => {
  switch (state) {
    case "active":
      return {
        tabFill: BLUE_DARK,
        tabCenter: BLUE_BRIGHT,
        tabBorder: BLUE_BRIGHT,
        iconColor: BLUE_BRIGHT,
        iconBg: BLUE_DARK,
        iconBorder: BLUE_BRIGHT,
        textColor: TEXT_BRIGHT,
        valueColor: TEXT_BRIGHT,
      };
    case "alarm":
      return {
        tabFill: RED_DARK,
        tabCenter: RED_BRIGHT,
        tabBorder: RED_BRIGHT,
        iconColor: RED_BRIGHT,
        iconBg: RED_DARK,
        iconBorder: RED_BRIGHT,
        textColor: RED_BRIGHT,
        valueColor: RED_BRIGHT,
      };
    case "excluded":
      return {
        tabFill: RED_DARK,
        tabCenter: RED_BRIGHT,
        tabBorder: RED_BRIGHT,
        iconColor: RED_BRIGHT,
        iconBg: RED_DARK,
        iconBorder: RED_BRIGHT,
        textColor: RED_BRIGHT,
        valueColor: RED_BRIGHT,
      };
    case "inactive":
      return {
        tabFill: DIM_FILL,
        tabCenter: DIM_BORDER,
        tabBorder: DIM_BORDER,
        iconColor: TEXT_DIM,
        iconBg: DIM_FILL,
        iconBorder: TEXT_DIM,
        textColor: TEXT_DIM,
        valueColor: TEXT_DIM,
      };
  }
};

export const resolveTraceColor = (
  flowing: boolean,
  excluded: boolean,
): string => {
  if (excluded) return RED_DARK;
  if (flowing) return BLUE_TRACE;
  return DIM_TRACE;
};

export const EXCLUDED_MARKER_COLOR = RED_BRIGHT;

export const deriveNodeState = (
  nodeId: string,
  activeNodes: string[],
  excludedNodes: string[],
  alarmNodes: string[] = [],
): NodeState => {
  if (excludedNodes.includes(nodeId)) return "excluded";
  if (alarmNodes.includes(nodeId)) return "alarm";
  if (activeNodes.includes(nodeId)) return "active";
  return "inactive";
};
