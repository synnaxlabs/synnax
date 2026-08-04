// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "./SchematicStyleShowcase.css";

import { type CSSProperties, Fragment, type ReactElement } from "react";

import { Flex } from "@/flex";
import { Text } from "@/text";

// Value study. Reference row reproduces the shipped value; every other row is
// the quiet chassis with channel color allowed in exactly one place. Columns
// show neutral, colored, warning, and critical (soft-fill tint alarms).

const TEAL = {
  "--schx-color": "#3fb6c2",
  "--schx-rgb": "63, 182, 194",
} as CSSProperties;

type AlarmState = "rest" | "warn" | "crit";

interface VariantSpec {
  key: string;
  name: string;
  note: string;
  modifier: string;
}

const VARIANTS: VariantSpec[] = [
  {
    key: "march",
    name: "March branch",
    note: "The sy-3882 value: 1px channel border + 6% tint fill when colored.",
    modifier: "schxv--march",
  },
  {
    key: "plain",
    name: "Quiet",
    note: "No channel color at all. Alarms are the only color.",
    modifier: "",
  },
  {
    key: "tick",
    name: "Quiet + tick",
    note: "Channel color as a 2px left edge.",
    modifier: "schxv--tick",
  },
  {
    key: "reading",
    name: "Quiet + reading",
    note: "The number itself takes the channel color.",
    modifier: "schxv--reading",
  },
  {
    key: "units",
    name: "Quiet + units",
    note: "Units text takes the channel color.",
    modifier: "schxv--units",
  },
  {
    key: "capsule",
    name: "Quiet + capsule",
    note: "Units in a neutral capsule. Echo of the old pill, defanged.",
    modifier: "schxv--capsule",
  },
  {
    key: "capsule-tinted",
    name: "Quiet + tinted capsule",
    note: "Capsule takes a soft tint of the channel color.",
    modifier: "schxv--capsule schxv--capsule-tinted",
  },
];

const ALARM_CLASS: Record<AlarmState, string> = {
  rest: "",
  warn: "schxv--warn",
  crit: "schxv--crit",
};

const QuietValue = ({
  modifier,
  alarm = "rest",
  colored = false,
  reading = "1250.2",
  units = "psi",
}: {
  modifier: string;
  alarm?: AlarmState;
  colored?: boolean;
  reading?: string;
  units?: string;
}): ReactElement => (
  <div
    className={`schxv ${modifier} ${colored ? "schxv--colored" : ""} ${ALARM_CLASS[alarm]}`}
    style={colored ? TEAL : undefined}
  >
    <span className="schxv__reading">{reading}</span>
    <span className="schxv__units">{units}</span>
  </div>
);

interface StateSpec {
  label: string;
  hex?: string;
  rgb?: string;
}

const STATES: StateSpec[] = [
  { label: "IDLE" },
  { label: "RUNNING", hex: "#4caf7d", rgb: "76, 175, 125" },
  { label: "WARNING", hex: "#e2a13c", rgb: "226, 161, 60" },
  { label: "FAULT", hex: "#e0574a", rgb: "224, 87, 74" },
];

const STATE_VARIANTS: VariantSpec[] = [
  {
    key: "twin",
    name: "Value twin",
    note: "The new value's colored recipe: 1px state border + 6% tint.",
    modifier: "schxs--twin",
  },
  {
    key: "tint",
    name: "Soft tint",
    note: "12% fill and colored text. Border stays neutral.",
    modifier: "schxs--tint",
  },
  {
    key: "dot",
    name: "Dot",
    note: "Quietest. Color is just a marker; chrome matches the value.",
    modifier: "schxs--dot",
  },
  {
    key: "pill",
    name: "Pill",
    note: "Borderless soft-fill capsule with dot.",
    modifier: "schxs--pill",
  },
];

const HUES: StateSpec[] = [
  { label: "Neutral" },
  { label: "Teal", hex: "#3fb6c2", rgb: "63, 182, 194" },
  { label: "Blue", hex: "#3774d0", rgb: "55, 116, 208" },
  { label: "Orange", hex: "#e28e45", rgb: "226, 142, 69" },
];

const SETPOINT_VARIANTS: VariantSpec[] = [
  {
    key: "current",
    name: "Current",
    note: "Shipped today: colored border, solid colored Set.",
    modifier: "schxp--current",
  },
  {
    key: "quiet",
    name: "Quiet",
    note: "Fully neutral. Color ignored everywhere.",
    modifier: "schxp--quiet",
  },
  {
    key: "tinted",
    name: "Tinted action",
    note: "Neutral chassis; SET takes a soft tint of the color.",
    modifier: "schxp--tinted",
  },
  {
    key: "twin",
    name: "Value twin",
    note: "Colored chassis; SET muted to a soft tint of the color.",
    modifier: "schxp--twin",
  },
  {
    key: "hybrid",
    name: "Twin + neutral",
    note: "Colored chassis; SET fully neutral gray.",
    modifier: "schxp--hybrid",
  },
];

const SetpointChip = ({
  modifier,
  hue,
}: {
  modifier: string;
  hue: StateSpec;
}): ReactElement => (
  <div
    className={`schxp ${modifier} ${hue.hex != null ? "schxp--colored" : ""}`}
    style={stateStyle(hue)}
  >
    <span className="schxp__field">
      500.0<span className="schxp__units">psi</span>
    </span>
    <button className="schxp__go">SET</button>
  </div>
);

const stateStyle = (s: StateSpec): CSSProperties | undefined =>
  s.hex != null
    ? ({ "--schx-color": s.hex, "--schx-rgb": s.rgb } as CSSProperties)
    : undefined;

const StateChip = ({
  modifier,
  state,
}: {
  modifier: string;
  state: StateSpec;
}): ReactElement => (
  <div
    className={`schxs ${modifier} ${state.hex != null ? "schxs--colored" : ""}`}
    style={stateStyle(state)}
  >
    <span className="schxs__dot" />
    <span>{state.label}</span>
  </div>
);

const CurrentState = ({ state }: { state: StateSpec }): ReactElement => (
  <div
    className={`schxs-current ${state.hex != null ? "schxs--colored" : ""}`}
    style={stateStyle(state)}
  >
    {state.label}
  </div>
);

const CurrentValue = ({
  alarm = "rest",
  colored = false,
}: {
  alarm?: AlarmState;
  colored?: boolean;
}): ReactElement => (
  <div
    className={`schx-current ${alarm !== "rest" ? `schx-current--${alarm}` : ""}`}
    style={colored && alarm === "rest" ? TEAL : undefined}
  >
    <span className="schx-current__reading">1250.2</span>
    <span className="schx-current__units">psi</span>
  </div>
);

export const SchematicStyleShowcase = (): ReactElement => (
  <Flex.Box y gap="large" style={{ padding: "3rem" }}>
    <Text.Text level="p" color={9}>
      Verdicts so far: value takes the quiet chassis, alarms take soft-fill tints,
      valves keep their production look, ink is out. Each row below allows channel color
      in exactly one place.
    </Text.Text>
    <div className="schx-canvas">
      <div className="schx-grid">
        <span />
        <span className="schx-grid__head">Neutral</span>
        <span className="schx-grid__head">Channel color</span>
        <span className="schx-grid__head">Warning</span>
        <span className="schx-grid__head">Critical</span>

        <div className="schx-grid__name">
          <span>Current</span>
          <span>Shipped today, for reference.</span>
        </div>
        <CurrentValue />
        <CurrentValue colored />
        <CurrentValue alarm="warn" />
        <CurrentValue alarm="crit" />

        {VARIANTS.map(({ key, name, note, modifier }) => (
          <Fragment key={key}>
            <div className="schx-grid__name">
              <span>{name}</span>
              <span>{note}</span>
            </div>
            <QuietValue modifier={modifier} />
            <QuietValue modifier={modifier} colored />
            <QuietValue modifier={modifier} alarm="warn" />
            <QuietValue modifier={modifier} alarm="crit" />
          </Fragment>
        ))}
      </div>
    </div>
    <Text.Text level="h4">State indicator</Text.Text>
    <Text.Text level="p" color={9}>
      Columns are matched states; IDLE has no configured color. Each row spends the
      state color differently, quietest at the bottom of the loud ones.
    </Text.Text>
    <div className="schx-canvas">
      <div className="schx-grid">
        <span />
        {STATES.map(({ label }) => (
          <span key={label} className="schx-grid__head">
            {label}
          </span>
        ))}

        <div className="schx-grid__name">
          <span>Current</span>
          <span>Shipped today: 2px border, solid state fill.</span>
        </div>
        {STATES.map((s) => (
          <CurrentState key={s.label} state={s} />
        ))}

        {STATE_VARIANTS.map(({ key, name, note, modifier }) => (
          <Fragment key={key}>
            <div className="schx-grid__name">
              <span>{name}</span>
              <span>{note}</span>
            </div>
            {STATES.map((s) => (
              <StateChip key={s.label} modifier={modifier} state={s} />
            ))}
          </Fragment>
        ))}
      </div>
    </div>
    <Text.Text level="h4">Setpoint</Text.Text>
    <Text.Text level="p" color={9}>
      Columns are symbol colors. Rows vary the chassis and how loud the SET action is.
      Hover and press the buttons; interaction states are part of the design.
    </Text.Text>
    <div className="schx-canvas">
      <div className="schx-grid">
        <span />
        {HUES.map(({ label }) => (
          <span key={label} className="schx-grid__head">
            {label}
          </span>
        ))}

        {SETPOINT_VARIANTS.map(({ key, name, note, modifier }) => (
          <Fragment key={key}>
            <div className="schx-grid__name">
              <span>{name}</span>
              <span>{note}</span>
            </div>
            {HUES.map((h) => (
              <SetpointChip key={h.label} modifier={modifier} hue={h} />
            ))}
          </Fragment>
        ))}
      </div>
    </div>
  </Flex.Box>
);
