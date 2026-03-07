// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/showcase/SchematicSymbolShowcase.css";

import { color } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { Flex } from "@/flex";
import * as Primitives from "@/schematic/symbol/Primitives";
import {
  GROUPS,
  REGISTRY,
  type Spec,
  type Variant,
} from "@/schematic/symbol/registry";
import { Text } from "@/text";
import { Theming } from "@/theming";

const rgba = (c: color.Crude, a: number): string => {
  const [r, g, b] = color.construct(c);
  return `rgba(${r}, ${g}, ${b}, ${a})`;
};

// ---- Experimental Value Variations ----

const ValueA = (): ReactElement => (
  <div className="exp-value exp-value--a">
    <span className="exp-value__number">50.00</span>
    <span className="exp-value__units">psi</span>
  </div>
);

const ValueB = (): ReactElement => (
  <div className="exp-value exp-value--b">
    <span className="exp-value__number">50.00</span>
    <span className="exp-value__units">psi</span>
  </div>
);

const ValueC = (): ReactElement => (
  <div className="exp-value exp-value--c">
    <span className="exp-value__number">50.00</span>
    <span className="exp-value__units">psi</span>
  </div>
);

// Color variants of value to show how user-assigned color affects it

const ValueAColored = ({ c }: { c: string }): ReactElement => (
  <div className="exp-value exp-value--a" style={{ borderColor: c }}>
    <span className="exp-value__number">50.00</span>
    <span className="exp-value__units">psi</span>
  </div>
);

const ValueBColored = ({ c }: { c: string }): ReactElement => (
  <div className="exp-value exp-value--b" style={{ borderColor: c }}>
    <span className="exp-value__number">50.00</span>
    <span className="exp-value__units">psi</span>
  </div>
);

const ValueCColored = ({ c }: { c: string }): ReactElement => (
  <div className="exp-value exp-value--c" style={{ background: rgba(c, 0.08) }}>
    <span className="exp-value__number">50.00</span>
    <span className="exp-value__units">psi</span>
  </div>
);

const ValueD = (): ReactElement => (
  <div className="exp-value exp-value--d">
    <span className="exp-value__number">50.00</span>
    <span className="exp-value__units">psi</span>
  </div>
);

const ValueDColored = ({ c }: { c: string }): ReactElement => (
  <div
    className="exp-value exp-value--d"
    style={{ borderColor: c, background: rgba(c, 0.08) }}
  >
    <span className="exp-value__number">50.00</span>
    <span
      className="exp-value__units"
      style={{ borderLeftColor: rgba(c, 0.3) }}
    >
      psi
    </span>
  </div>
);

// ---- Experimental Light Variations ----

type ExpSize = "sm" | "md" | "lg";

const sizeClass = (size: ExpSize): string =>
  size === "md" ? "" : ` exp-light--${size}`;

const stateSizeClass = (size: ExpSize): string =>
  size === "md" ? "" : ` exp-state--${size}`;

interface LightExpProps {
  label: string;
  enabled: boolean;
  c: string;
  size?: ExpSize;
}

const LightA = ({ label, enabled, c, size = "md" }: LightExpProps): ReactElement => {
  const borderColor = rgba(c, enabled ? 1 : 0.3);
  const background = rgba(c, enabled ? 0.9 : 0.08);
  const textColor = enabled
    ? color.cssString(color.pickByContrast(c, "#FEFEFE", "#050505"))
    : rgba(c, 0.6);
  return (
    <div
      className={`exp-light exp-light--a${sizeClass(size)}`}
      style={{ border: `1px solid ${borderColor}`, background }}
    >
      <span className="exp-light__label" style={{ color: textColor }}>
        {label}
      </span>
    </div>
  );
};

const LightB = ({
  label,
  enabled,
  c,
  size = "md",
}: LightExpProps): ReactElement => {
  const borderColor = rgba(c, enabled ? 1 : 0.3);
  const background = rgba(c, enabled ? 0.9 : 0.08);
  const textColor = enabled
    ? color.cssString(color.pickByContrast(c, "#FEFEFE", "#050505"))
    : rgba(c, 0.6);
  const glow: CSSProperties = enabled
    ? { boxShadow: `0 0 8px 1px ${rgba(c, 0.35)}` }
    : {};
  return (
    <div
      className={`exp-light exp-light--b${sizeClass(size)}`}
      style={{ border: `1px solid ${borderColor}`, background, ...glow }}
    >
      <span className="exp-light__label" style={{ color: textColor }}>
        {label}
      </span>
    </div>
  );
};

const LightC = ({
  enabled,
  c,
  size = "md",
}: Omit<LightExpProps, "label">): ReactElement => {
  const border = `1px solid ${rgba(c, enabled ? 0.8 : 0.25)}`;
  const background = rgba(c, enabled ? 0.85 : 0.06);
  const glow: CSSProperties = enabled
    ? { boxShadow: `0 0 8px 2px ${rgba(c, 0.3)}` }
    : {};
  return (
    <div
      className={`exp-light exp-light--c${sizeClass(size)}`}
      style={{ border, background, ...glow }}
    />
  );
};

// ---- Experimental State Indicator Variations ----

interface StateExpProps {
  label: string;
  stateColor: string;
  isAbnormal: boolean;
  size?: ExpSize;
}

const StateA = ({
  label,
  stateColor,
  isAbnormal,
  size = "md",
}: StateExpProps): ReactElement => {
  const background = isAbnormal
    ? rgba(stateColor, 0.1)
    : "var(--pluto-gray-l2)";
  const border = isAbnormal
    ? `1px solid ${rgba(stateColor, 0.3)}`
    : "1px solid var(--pluto-gray-l4)";
  const dotColor = isAbnormal
    ? rgba(stateColor, 1)
    : "var(--pluto-gray-l7)";
  const textColor = isAbnormal
    ? rgba(stateColor, 1)
    : "var(--pluto-gray-l9)";
  return (
    <div
      className={`exp-state exp-state--a${stateSizeClass(size)}`}
      style={{ background, border }}
    >
      <div className="exp-state__dot" style={{ background: dotColor }} />
      <span className="exp-state__label" style={{ color: textColor }}>
        {label}
      </span>
    </div>
  );
};

const StateB = ({
  label,
  stateColor,
  isAbnormal,
  size = "md",
}: StateExpProps): ReactElement => {
  const borderColor = isAbnormal
    ? rgba(stateColor, 0.6)
    : "var(--pluto-gray-l5)";
  const background = isAbnormal
    ? rgba(stateColor, 0.15)
    : "var(--pluto-gray-l2)";
  const textColor = isAbnormal
    ? rgba(stateColor, 1)
    : "var(--pluto-gray-l9)";
  return (
    <div
      className={`exp-state exp-state--b${stateSizeClass(size)}`}
      style={{ border: `1px solid ${borderColor}`, background }}
    >
      <span className="exp-state__label" style={{ color: textColor }}>
        {label}
      </span>
    </div>
  );
};

const StateC = ({
  label,
  stateColor,
  isAbnormal,
  size = "md",
}: StateExpProps): ReactElement => {
  const borderColor = isAbnormal
    ? rgba(stateColor, 0.6)
    : "var(--pluto-gray-l5)";
  const background = isAbnormal
    ? rgba(stateColor, 0.08)
    : "var(--pluto-gray-l2)";
  const dotColor = isAbnormal
    ? rgba(stateColor, 1)
    : "var(--pluto-gray-l7)";
  const textColor = isAbnormal
    ? rgba(stateColor, 1)
    : "var(--pluto-gray-l9)";
  return (
    <div
      className={`exp-state exp-state--c${stateSizeClass(size)}`}
      style={{ border: `1px solid ${borderColor}`, background }}
    >
      <div className="exp-state__dot" style={{ background: dotColor }} />
      <span className="exp-state__label" style={{ color: textColor }}>
        {label}
      </span>
    </div>
  );
};

// ---- Section helpers ----

const VariationLabel = ({ children }: { children: string }): ReactElement => (
  <Text.Text level="small" color={8} weight={500}>
    {children}
  </Text.Text>
);

const SectionTitle = ({
  title,
  description,
}: {
  title: string;
  description: string;
}): ReactElement => (
  <Flex.Box y gap="small">
    <Text.Text level="h4">{title}</Text.Text>
    <Text.Text level="small" color={9}>
      {description}
    </Text.Text>
  </Flex.Box>
);

const CurrentLabel = (): ReactElement => (
  <Text.Text
    level="small"
    color={8}
    weight={500}
    style={{ fontStyle: "italic" }}
  >
    Current
  </Text.Text>
);

// ---- Main Showcase Sections ----

const ValueExperiments = (): ReactElement => {
  const theme = Theming.use();
  const accentColor = "#3774D0";
  const errorColor = "#F5242E";
  return (
    <Flex.Box y gap="huge" style={{ padding: "3rem" }} grow>
      <SectionTitle
        title="Value Display"
        description="The numeric value should be the hero. Units should be quiet and subordinate. Borders should feel refined, not heavy."
      />

      {/* Default color comparison */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Default (no user color)</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Primitives.Value
              color={theme.colors.gray.l11}
              dimensions={{ width: 60, height: 25 }}
              units="psi"
            >
              <Text.Text>50.00</Text.Text>
            </Primitives.Value>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Minimal</VariationLabel>
            <ValueA />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Divided</VariationLabel>
            <ValueB />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Borderless</VariationLabel>
            <ValueC />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>D: Borderless + Divided</VariationLabel>
            <ValueD />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* With accent color */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>
          With user color (blue)
        </Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Primitives.Value
              color={accentColor}
              dimensions={{ width: 60, height: 25 }}
              units="psi"
            >
              <Text.Text>50.00</Text.Text>
            </Primitives.Value>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Minimal</VariationLabel>
            <ValueAColored c={accentColor} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Divided</VariationLabel>
            <ValueBColored c={accentColor} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Borderless</VariationLabel>
            <ValueCColored c={accentColor} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>D: Borderless + Divided</VariationLabel>
            <ValueDColored c={accentColor} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* With error/alarm color */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>
          With alarm color (red)
        </Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Primitives.Value
              color={errorColor}
              dimensions={{ width: 60, height: 25 }}
              units="psi"
            >
              <Text.Text>50.00</Text.Text>
            </Primitives.Value>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Minimal</VariationLabel>
            <ValueAColored c={errorColor} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Divided</VariationLabel>
            <ValueBColored c={errorColor} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Borderless</VariationLabel>
            <ValueCColored c={errorColor} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>D: Borderless + Divided</VariationLabel>
            <ValueDColored c={errorColor} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

const LightExperiments = (): ReactElement => {
  const green = "#50C878";
  const red = "#F5242E";
  const blue = "#3774D0";
  const theme = Theming.use();
  return (
    <Flex.Box y gap="huge" style={{ padding: "3rem" }} grow>
      <SectionTitle
        title="Light / Indicator"
        description="Binary on/off indicator. Off should fade into the background. On should draw the eye. Apollo-inspired rectangular variants include a text label."
      />

      {/* Green indicator */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Green indicator</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Flex.Box x gap="large" align="center">
              <Primitives.Light color={green} enabled={false} />
              <Primitives.Light color={green} enabled />
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Apollo</VariationLabel>
            <Flex.Box x gap="large" align="center">
              <LightA label="PRESS" enabled={false} c={green} />
              <LightA label="PRESS" enabled c={green} />
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Apollo + Glow</VariationLabel>
            <Flex.Box x gap="large" align="center">
              <LightB label="PRESS" enabled={false} c={green} />
              <LightB label="PRESS" enabled c={green} />
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Refined Circle</VariationLabel>
            <Flex.Box x gap="large" align="center">
              <LightC enabled={false} c={green} />
              <LightC enabled c={green} />
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* Red indicator */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Red indicator (alarm)</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Flex.Box x gap="large" align="center">
              <Primitives.Light color={red} enabled={false} />
              <Primitives.Light color={red} enabled />
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Apollo</VariationLabel>
            <Flex.Box x gap="large" align="center">
              <LightA label="ALARM" enabled={false} c={red} />
              <LightA label="ALARM" enabled c={red} />
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Apollo + Glow</VariationLabel>
            <Flex.Box x gap="large" align="center">
              <LightB label="ALARM" enabled={false} c={red} />
              <LightB label="ALARM" enabled c={red} />
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Refined Circle</VariationLabel>
            <Flex.Box x gap="large" align="center">
              <LightC enabled={false} c={red} />
              <LightC enabled c={red} />
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* Blue indicator */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Blue indicator</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Flex.Box x gap="large" align="center">
              <Primitives.Light color={blue} enabled={false} />
              <Primitives.Light color={blue} enabled />
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Apollo</VariationLabel>
            <Flex.Box x gap="large" align="center">
              <LightA label="POWER" enabled={false} c={blue} />
              <LightA label="POWER" enabled c={blue} />
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Apollo + Glow</VariationLabel>
            <Flex.Box x gap="large" align="center">
              <LightB label="POWER" enabled={false} c={blue} />
              <LightB label="POWER" enabled c={blue} />
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Refined Circle</VariationLabel>
            <Flex.Box x gap="large" align="center">
              <LightC enabled={false} c={blue} />
              <LightC enabled c={blue} />
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* Size variants */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Size variants (green, on)</Text.Text>
        <Flex.Box x gap="huge" align="end" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Small</VariationLabel>
            <LightA label="PRESS" enabled c={green} size="sm" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Medium</VariationLabel>
            <LightA label="PRESS" enabled c={green} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Large</VariationLabel>
            <LightA label="PRESS" enabled c={green} size="lg" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Small</VariationLabel>
            <LightB label="PRESS" enabled c={green} size="sm" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Medium</VariationLabel>
            <LightB label="PRESS" enabled c={green} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Large</VariationLabel>
            <LightB label="PRESS" enabled c={green} size="lg" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Small</VariationLabel>
            <LightC enabled c={green} size="sm" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Medium</VariationLabel>
            <LightC enabled c={green} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Large</VariationLabel>
            <LightC enabled c={green} size="lg" />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

const StateIndicatorExperiments = (): ReactElement => {
  const green = "#50C878";
  const red = "#F5242E";
  const amber = "#F4CA25";
  const theme = Theming.use();
  return (
    <Flex.Box y gap="huge" style={{ padding: "3rem" }} grow>
      <SectionTitle
        title="State Indicator"
        description="Multi-state status display. Should be visually distinct from the value display. Normal states should be muted, abnormal states should draw attention."
      />

      {/* Normal state */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Normal state (Running)</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Primitives.StateIndicator
              matchedOptionKey="1"
              options={[{ key: "1", name: "Running", value: 1, color: green }]}
              color={theme.colors.gray.l11}
            />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Status Pill</VariationLabel>
            <StateA label="Running" stateColor={green} isAbnormal={false} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Labeled Light</VariationLabel>
            <StateB label="Running" stateColor={green} isAbnormal={false} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Rectangle + Dot</VariationLabel>
            <StateC label="Running" stateColor={green} isAbnormal={false} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* Fault state */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Abnormal state (Fault)</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Primitives.StateIndicator
              matchedOptionKey="1"
              options={[{ key: "1", name: "Fault", value: 1, color: red }]}
              color={theme.colors.gray.l11}
            />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Status Pill</VariationLabel>
            <StateA label="Fault" stateColor={red} isAbnormal />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Labeled Light</VariationLabel>
            <StateB label="Fault" stateColor={red} isAbnormal />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Rectangle + Dot</VariationLabel>
            <StateC label="Fault" stateColor={red} isAbnormal />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* Warning state */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Warning state</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Primitives.StateIndicator
              matchedOptionKey="1"
              options={[{ key: "1", name: "Warning", value: 1, color: amber }]}
              color={theme.colors.gray.l11}
            />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Status Pill</VariationLabel>
            <StateA label="Warning" stateColor={amber} isAbnormal />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Labeled Light</VariationLabel>
            <StateB label="Warning" stateColor={amber} isAbnormal />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Rectangle + Dot</VariationLabel>
            <StateC label="Warning" stateColor={amber} isAbnormal />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* Stopped (neutral) state */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Neutral state (Stopped)</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <CurrentLabel />
            <Primitives.StateIndicator
              matchedOptionKey="1"
              options={[
                {
                  key: "1",
                  name: "Stopped",
                  value: 1,
                  color: theme.colors.gray.l7,
                },
              ]}
              color={theme.colors.gray.l11}
            />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Status Pill</VariationLabel>
            <StateA
              label="Stopped"
              stateColor={color.cssString(theme.colors.gray.l7)}
              isAbnormal={false}
            />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Labeled Light</VariationLabel>
            <StateB
              label="Stopped"
              stateColor={color.cssString(theme.colors.gray.l7)}
              isAbnormal={false}
            />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Rectangle + Dot</VariationLabel>
            <StateC
              label="Stopped"
              stateColor={color.cssString(theme.colors.gray.l7)}
              isAbnormal={false}
            />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* Size variants */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Size variants (Fault, red)</Text.Text>
        <Flex.Box x gap="huge" align="end" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Small</VariationLabel>
            <StateC label="Fault" stateColor={red} isAbnormal size="sm" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Medium</VariationLabel>
            <StateC label="Fault" stateColor={red} isAbnormal />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Large</VariationLabel>
            <StateC label="Fault" stateColor={red} isAbnormal size="lg" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Small</VariationLabel>
            <StateA label="Fault" stateColor={red} isAbnormal size="sm" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Medium</VariationLabel>
            <StateA label="Fault" stateColor={red} isAbnormal />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Large</VariationLabel>
            <StateA label="Fault" stateColor={red} isAbnormal size="lg" />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

// ---- Experimental Button Variations ----

interface ButtonExpProps {
  label: string;
  c: string;
}

const ButtonA = ({ label, c }: ButtonExpProps): ReactElement => {
  const textColor = color.cssString(color.pickByContrast(c, "#FEFEFE", "#050505"));
  return (
    <button className="exp-button exp-button--a" style={{ background: c, color: textColor }}>
      {label}
    </button>
  );
};

const ButtonB = ({ label, c }: ButtonExpProps): ReactElement => (
  <button
    className="exp-button exp-button--b"
    style={{
      border: `1px solid ${c}`,
      background: rgba(c, 0.08),
      color: c,
    }}
  >
    {label}
  </button>
);

const ButtonC = ({ label, c }: ButtonExpProps): ReactElement => (
  <button
    className="exp-button exp-button--c"
    style={{
      border: `1px solid ${c}`,
      background: "transparent",
      color: c,
    }}
  >
    {label}
  </button>
);

// ---- Experimental Tank Variations ----

interface TankExpProps {
  c: color.Crude;
  width: number;
  height: number;
}

const TankA = ({ c, width, height }: TankExpProps): ReactElement => (
  <div
    className="exp-tank exp-tank--a"
    style={{
      width,
      height,
      borderColor: color.cssString(c),
      borderRadius: "2px 2px 2px 2px / 2px 2px 2px 2px",
    }}
  />
);

const TankB = ({ c, width, height }: TankExpProps): ReactElement => (
  <div
    className="exp-tank exp-tank--b"
    style={{
      width,
      height,
      borderColor: color.cssString(c),
      borderRadius: "4px 4px 50% 50% / 4px 4px 12% 12%",
      background: rgba(c, 0.05),
    }}
  />
);

const TankC = ({ c, width, height }: TankExpProps): ReactElement => (
  <div
    className="exp-tank exp-tank--c"
    style={{
      width,
      height,
      borderColor: color.cssString(c),
      borderRadius: "4px 4px 50% 50% / 4px 4px 18% 18%",
      background: rgba(c, 0.05),
    }}
  />
);

const TankD = ({ c, width, height }: TankExpProps): ReactElement => (
  <div
    className="exp-tank exp-tank--b"
    style={{
      width,
      height,
      borderColor: color.cssString(c),
      borderRadius: "4px 4px 50% 50% / 4px 4px 25% 25%",
      background: rgba(c, 0.05),
      borderTopWidth: 2,
    }}
  />
);

const TankE = ({ c, width, height }: TankExpProps): ReactElement => (
  <div
    className="exp-tank exp-tank--b"
    style={{
      width,
      height,
      borderColor: color.cssString(c),
      borderRadius: "50% 50% 50% 50% / 15% 15% 15% 15%",
      background: rgba(c, 0.05),
    }}
  />
);

const TankF = ({ c, width, height }: TankExpProps): ReactElement => (
  <div
    className="exp-tank exp-tank--b"
    style={{
      width,
      height,
      borderColor: color.cssString(c),
      borderRadius: "50% 50% 50% 50% / 15% 15% 15% 15%",
      background: rgba(c, 0.05),
      borderTopWidth: 2,
      borderBottomWidth: 2,
    }}
  />
);

const TankG = ({ c, width, height }: TankExpProps): ReactElement => (
  <div
    className="exp-tank exp-tank--c"
    style={{
      width,
      height,
      borderColor: color.cssString(c),
      borderRadius: "50% 50% 50% 50% / 12% 12% 12% 12%",
      background: rgba(c, 0.05),
      borderTopWidth: 1.5,
      borderBottomWidth: 1.5,
    }}
  />
);

// ---- Experimental Input/Setpoint Variations ----

interface InputExpProps {
  c: string;
  value: string;
  units?: string;
  actionLabel: string;
}

const InputA = ({ c, value, units, actionLabel }: InputExpProps): ReactElement => {
  const textColor = color.cssString(color.pickByContrast(c, "#FEFEFE", "#050505"));
  return (
    <div className="exp-input exp-input--a">
      <input className="exp-input__field" value={value} readOnly />
      {units != null && <span className="exp-input__units">{units}</span>}
      <button
        className="exp-input__action"
        style={{ background: c, color: textColor }}
      >
        {actionLabel}
      </button>
    </div>
  );
};

const InputB = ({ c, value, units, actionLabel }: InputExpProps): ReactElement => {
  const textColor = color.cssString(color.pickByContrast(c, "#FEFEFE", "#050505"));
  return (
    <div className="exp-input exp-input--b">
      <input className="exp-input__field" value={value} readOnly />
      {units != null && <span className="exp-input__units">{units}</span>}
      <button
        className="exp-input__action"
        style={{ background: c, color: textColor }}
      >
        {actionLabel}
      </button>
    </div>
  );
};

// ---- Experimental Valve Variations ----

interface ValveExpProps {
  c: color.Crude;
  strokeWidth: number;
  fillOpacity: number;
}

const SolenoidValveExp = ({
  c,
  strokeWidth,
  fillOpacity,
}: ValveExpProps): ReactElement => {
  const stroke = color.cssString(c);
  const fill = fillOpacity > 0 ? rgba(c, fillOpacity) : "none";
  return (
    <div className="exp-valve">
      <svg width="87" height="69" viewBox="0 0 87 69">
        <path
          d="M43.5 48L6.35453 29.2035C4.35901 28.1937 2 29.6438 2 31.8803V64.1197C2 66.3562 4.35901 67.8063 6.35453 66.7965L43.5 48ZM43.5 48L80.6455 29.2035C82.641 28.1937 85 29.6438 85 31.8803V64.1197C85 66.3562 82.641 67.8063 80.6455 66.7965L43.5 48Z"
          stroke={stroke}
          strokeWidth={strokeWidth}
          fill={fill}
        />
        <line
          x1={43.5}
          x2={43.5}
          y1={24.5333}
          y2={48}
          stroke={stroke}
          strokeWidth={strokeWidth}
        />
        <rect
          x="29"
          y="2"
          width="29"
          height="22.5333"
          rx="1"
          stroke={stroke}
          strokeWidth={strokeWidth}
          fill={fill}
        />
      </svg>
    </div>
  );
};

// ---- Main Showcase: Other Primitives ----

const ButtonExperiments = (): ReactElement => {
  const blue = "#3774D0";
  const green = "#50C878";
  const red = "#F5242E";
  return (
    <Flex.Box y gap="huge" style={{ padding: "3rem" }} grow>
      <SectionTitle
        title="Button"
        description="Interactive control for sending commands. Should feel clickable and purposeful without being visually heavy."
      />
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Blue</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Filled (current)</VariationLabel>
            <ButtonA label="Set" c={blue} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Outlined + Fill + Shadow</VariationLabel>
            <ButtonB label="Set" c={blue} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Outlined</VariationLabel>
            <ButtonC label="Set" c={blue} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Green</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Filled (current)</VariationLabel>
            <ButtonA label="Send" c={green} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Outlined + Fill + Shadow</VariationLabel>
            <ButtonB label="Send" c={green} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Outlined</VariationLabel>
            <ButtonC label="Send" c={green} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Red (alarm/stop)</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Filled (current)</VariationLabel>
            <ButtonA label="Stop" c={red} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Outlined + Fill + Shadow</VariationLabel>
            <ButtonB label="Stop" c={red} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: Outlined</VariationLabel>
            <ButtonC label="Stop" c={red} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

const TankExperiments = (): ReactElement => {
  const theme = Theming.use();
  const neutral = theme.colors.gray.l8;
  const blue = "#3774D0";
  const w = 65;
  const h = 110;
  return (
    <Flex.Box y gap="huge" style={{ padding: "3rem" }} grow>
      <SectionTitle
        title="Tank / Vessel"
        description="Should communicate 'vessel' not just 'box'. Exploring rounded bottoms, subtle fills, and top edge treatments."
      />
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Neutral</Text.Text>
        <Flex.Box x gap="huge" align="end" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Box (current)</VariationLabel>
            <TankA c={neutral} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Rounded bottom, subtle</VariationLabel>
            <TankB c={neutral} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: More rounded bottom</VariationLabel>
            <TankC c={neutral} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>D: Rounded + top lip</VariationLabel>
            <TankD c={neutral} width={w} height={h} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>With color (blue)</Text.Text>
        <Flex.Box x gap="huge" align="end" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Box (current)</VariationLabel>
            <TankA c={blue} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Rounded bottom</VariationLabel>
            <TankB c={blue} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>C: More rounded</VariationLabel>
            <TankC c={blue} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>D: Rounded + top lip</VariationLabel>
            <TankD c={blue} width={w} height={h} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>

      {/* Fully rounded variants */}
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Fully rounded (pressure vessel)</Text.Text>
        <Flex.Box x gap="huge" align="end" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>E: All-round, 1px</VariationLabel>
            <TankE c={neutral} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>F: All-round, thick ends</VariationLabel>
            <TankF c={neutral} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>G: All-round + inner shadow</VariationLabel>
            <TankG c={neutral} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>E: Blue</VariationLabel>
            <TankE c={blue} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>F: Blue, thick ends</VariationLabel>
            <TankF c={blue} width={w} height={h} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>G: Blue + inner shadow</VariationLabel>
            <TankG c={blue} width={w} height={h} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

const InputExperiments = (): ReactElement => {
  const blue = "#3774D0";
  const green = "#50C878";
  return (
    <Flex.Box y gap="huge" style={{ padding: "3rem" }} grow>
      <SectionTitle
        title="Setpoint / Input"
        description="Interactive data entry with send action. The input field and action button should feel unified but with clear hierarchy."
      />
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Setpoint (with units)</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Border + solid action</VariationLabel>
            <InputA c={blue} value="150.0" units="psi" actionLabel="Set" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Fill + shadow + solid action</VariationLabel>
            <InputB c={blue} value="150.0" units="psi" actionLabel="Set" />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Input (no units)</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>A: Border + solid action</VariationLabel>
            <InputA c={green} value="GO" actionLabel="Send" />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>B: Fill + shadow + solid action</VariationLabel>
            <InputB c={green} value="GO" actionLabel="Send" />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

const ValveExperiments = (): ReactElement => {
  const theme = Theming.use();
  const neutral = theme.colors.gray.l8;
  const blue = "#3774D0";
  return (
    <Flex.Box y gap="huge" style={{ padding: "3rem" }} grow>
      <SectionTitle
        title="Solenoid Valve (SVG)"
        description="Engineering symbol. Exploring stroke weight and interior fill treatments to add visual richness without becoming skeuomorphic."
      />
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>Neutral</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>2px / no fill (current)</VariationLabel>
            <SolenoidValveExp c={neutral} strokeWidth={2} fillOpacity={0} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>2px / 4% fill</VariationLabel>
            <SolenoidValveExp c={neutral} strokeWidth={2} fillOpacity={0.04} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>2px / 8% fill</VariationLabel>
            <SolenoidValveExp c={neutral} strokeWidth={2} fillOpacity={0.08} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>1.5px / 6% fill</VariationLabel>
            <SolenoidValveExp c={neutral} strokeWidth={1.5} fillOpacity={0.06} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box y gap="medium">
        <Text.Text level="p" weight={500}>With color (blue)</Text.Text>
        <Flex.Box x gap="huge" align="center" wrap>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>2px / no fill (current)</VariationLabel>
            <SolenoidValveExp c={blue} strokeWidth={2} fillOpacity={0} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>2px / 4% fill</VariationLabel>
            <SolenoidValveExp c={blue} strokeWidth={2} fillOpacity={0.04} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>2px / 8% fill</VariationLabel>
            <SolenoidValveExp c={blue} strokeWidth={2} fillOpacity={0.08} />
          </Flex.Box>
          <Flex.Box y gap="small" align="center">
            <VariationLabel>1.5px / 6% fill</VariationLabel>
            <SolenoidValveExp c={blue} strokeWidth={1.5} fillOpacity={0.06} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

// ---- Symbol Grid (existing) ----

const WIDE_SYMBOLS: Set<Variant> = new Set([
  "value",
  "setpoint",
  "input",
  "select",
  "stateIndicator",
  "textBox",
  "gauge",
]);

const SHOWCASE_DIMENSIONS = { width: 75, height: 120 };

const PREVIEW_OVERRIDES: Partial<Record<Variant, (props: any) => ReactElement>> = {
  tank: (props) => (
    <Primitives.Tank {...props} dimensions={SHOWCASE_DIMENSIONS} />
  ),
  cylinder: (props) => (
    <Primitives.Cylinder {...props} dimensions={SHOWCASE_DIMENSIONS} />
  ),
  box: (props) => (
    <Primitives.Tank
      {...props}
      dimensions={SHOWCASE_DIMENSIONS}
      borderRadius={0}
    />
  ),
};

const SymbolCell = ({ spec }: { spec: Spec }): ReactElement => {
  const theme = Theming.use();
  const props = spec.defaultProps(theme);
  const override = PREVIEW_OVERRIDES[spec.key];
  const wide = WIDE_SYMBOLS.has(spec.key);
  const tall = override != null;
  return (
    <Flex.Box
      y
      align="center"
      justify="center"
      gap="small"
      style={{
        padding: "2rem",
        minWidth: wide ? "20rem" : "12rem",
        minHeight: tall ? "18rem" : "10rem",
      }}
    >
      <Flex.Box
        align="center"
        justify="center"
        style={{ flex: 1, minHeight: "5rem" }}
      >
        {override != null ? override(props) : <spec.Preview {...(props as any)} />}
      </Flex.Box>
      <Text.Text level="small" color={9} style={{ textAlign: "center" }}>
        {spec.name}
      </Text.Text>
    </Flex.Box>
  );
};

const GroupSection = ({
  groupKey,
}: {
  groupKey: string;
}): ReactElement | null => {
  const group = GROUPS.find((g) => g.key === groupKey);
  if (group == null) return null;
  const specs = group.symbols
    .map((variant) => REGISTRY[variant])
    .filter((s): s is Spec => s != null);
  if (specs.length === 0) return null;
  const GroupIcon = group.Icon;
  return (
    <Flex.Box y gap="large">
      <Flex.Box y gap="small">
        <Flex.Box x gap="small" align="center">
          <GroupIcon />
          <Text.Text level="h4">{group.name}</Text.Text>
        </Flex.Box>
        <Text.Text level="small" color={9}>
          {specs.length} symbols
        </Text.Text>
      </Flex.Box>
      <Flex.Box x wrap gap="medium">
        {specs.map((spec) => (
          <SymbolCell key={spec.key} spec={spec} />
        ))}
      </Flex.Box>
    </Flex.Box>
  );
};

// ---- Top-Level Showcase ----

export const SchematicSymbolShowcase = (): ReactElement => (
  <Flex.Box y gap="huge">
    {/* Experimental variations */}
    <Flex.Box y background={1} rounded={2} pack>
      <Flex.Box y bordered sharp grow>
        <ValueExperiments />
      </Flex.Box>
      <Flex.Box y bordered sharp grow>
        <LightExperiments />
      </Flex.Box>
      <Flex.Box y bordered sharp grow>
        <StateIndicatorExperiments />
      </Flex.Box>
    </Flex.Box>

    {/* Other primitive experiments */}
    <Flex.Box y background={1} rounded={2} pack>
      <Flex.Box y bordered sharp grow>
        <ButtonExperiments />
      </Flex.Box>
      <Flex.Box y bordered sharp grow>
        <TankExperiments />
      </Flex.Box>
      <Flex.Box y bordered sharp grow>
        <InputExperiments />
      </Flex.Box>
      <Flex.Box y bordered sharp grow>
        <ValveExperiments />
      </Flex.Box>
    </Flex.Box>

    {/* Full symbol grid */}
    <Flex.Box y background={1} rounded={2} pack>
      <Flex.Box y bordered style={{ padding: "3rem" }} rounded gap="huge">
        <SectionTitle
          title="All Symbols"
          description="Complete symbol grid organized by category."
        />
        {GROUPS.map((group) => (
          <GroupSection key={group.key} groupKey={group.key} />
        ))}
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);
