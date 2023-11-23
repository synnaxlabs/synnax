// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type xy } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { Primitives } from "@/vis/pid/symbols/primitives";
import { Toggle } from "@/vis/toggle";

import { Labeled, type LabelExtensionProps } from "./Labeled";

export type SymbolProps<P extends object> = P & {
  position: xy.XY;
  selected: boolean;
  editable: boolean;
  onChange: (value: P) => void;
};

export interface ThreeWayValveProps
  extends Primitives.ThreeWayValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
}

export const ThreeWayValve = Aether.wrap<SymbolProps<ThreeWayValveProps>>(
  "ThreeWayValve",
  ({
    aetherKey,
    label,
    editable: _,
    selected,
    onChange,
    source,
    sink,
    ...props
  }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <Primitives.ThreeWayValve
          enabled={enabled}
          triggered={triggered}
          onClick={toggle}
          {...props}
        />
      </Labeled>
    );
  },
);

export const ThreeWayValvePreview = (props: ThreeWayValveProps): ReactElement => (
  <Primitives.ThreeWayValve {...props} />
);

export interface ValveProps
  extends Primitives.ValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
}

export const Valve = Aether.wrap<SymbolProps<ValveProps>>(
  "Valve",
  ({ aetherKey, label, onChange, source, sink, ...props }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey, source, sink });
    return (
      <Labeled {...label} onChange={onChange}>
        <Primitives.Valve
          enabled={enabled}
          triggered={triggered}
          onClick={toggle}
          {...props}
        />
      </Labeled>
    );
  },
);

export const ValvePreview = (props: ValveProps): ReactElement => (
  <Primitives.Valve {...props} />
);

export interface SolenoidValveProps
  extends Primitives.SolenoidValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
}

export const SolenoidValve = Aether.wrap<SymbolProps<SolenoidValveProps>>(
  "SolenoidValve",
  ({ aetherKey, label, onChange, ...props }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey });
    return (
      <Labeled {...label} onChange={onChange}>
        <Primitives.SolenoidValve
          enabled={enabled}
          triggered={triggered}
          onClick={toggle}
          {...props}
        />
      </Labeled>
    );
  },
);

export const SolenoidValvePreview = (props: SolenoidValveProps): ReactElement => (
  <Primitives.SolenoidValve {...props} />
);

export interface FourWayValveProps
  extends Primitives.FourWayValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
}

export const FourWayValve = Aether.wrap<SymbolProps<FourWayValveProps>>(
  "FourWayValve",
  ({ aetherKey, label, onChange, ...props }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey });
    return (
      <Labeled {...label} onChange={onChange}>
        <Primitives.FourWayValve
          enabled={enabled}
          triggered={triggered}
          onClick={toggle}
          {...props}
        />
      </Labeled>
    );
  },
);

export const FourWayValvePreview = (props: FourWayValveProps): ReactElement => (
  <Primitives.FourWayValve {...props} />
);

export interface AngledValveProps
  extends Primitives.AngledValveProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
}

export const AngledValve = Aether.wrap<SymbolProps<AngledValveProps>>(
  "AngleValve",
  ({ aetherKey, label, onChange, ...props }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey });
    return (
      <Labeled {...label} onChange={onChange}>
        <Primitives.AngledValve
          enabled={enabled}
          triggered={triggered}
          onClick={toggle}
          {...props}
        />
      </Labeled>
    );
  },
);

export const AngledValvePreview = (props: AngledValveProps): ReactElement => (
  <Primitives.AngledValve {...props} />
);

export interface PumpProps
  extends Primitives.PumpProps,
    Omit<Toggle.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
}

export const Pump = Aether.wrap<SymbolProps<PumpProps>>(
  "Pump",
  ({ aetherKey, label, onChange, ...props }): ReactElement => {
    const { enabled, triggered, toggle } = Toggle.use({ aetherKey });
    return (
      <Labeled {...label} onChange={onChange}>
        <Primitives.Pump
          enabled={enabled}
          triggered={triggered}
          onClick={toggle}
          {...props}
        />
      </Labeled>
    );
  },
);

export const PumpPreview = (props: PumpProps): ReactElement => (
  <Primitives.Pump {...props} />
);

export interface TankProps extends Primitives.TankProps {
  label?: LabelExtensionProps;
}

export const Tank = Aether.wrap<SymbolProps<TankProps>>(
  "Tank",
  ({ aetherKey, label, onChange, ...props }): ReactElement => {
    return (
      <Labeled {...label} onChange={onChange}>
        <Primitives.Tank {...props} />
      </Labeled>
    );
  },
);

export const TankPreview = (props: TankProps): ReactElement => (
  <Primitives.Tank {...props} dimensions={{ width: 30, height: 60 }} />
);

export interface ReliefValveProps extends Primitives.ReliefValveProps {
  label?: LabelExtensionProps;
}

export const ReliefValve = ({
  label,
  onChange,
  ...props
}: SymbolProps<ReliefValveProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.ReliefValve {...props} />
    </Labeled>
  );
};

export const ReliefValvePreview = (props: ReliefValveProps): ReactElement => (
  <Primitives.ReliefValve {...props} />
);

export interface RegulatorProps extends Primitives.RegulatorProps {
  label?: LabelExtensionProps;
}

export const Regulator = ({
  label,
  onChange,
  ...props
}: SymbolProps<RegulatorProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.Regulator {...props} />
    </Labeled>
  );
};

export const RegulatorPreview = (props: RegulatorProps): ReactElement => (
  <Primitives.Regulator {...props} />
);

export interface BurstDiscProps extends Primitives.BurstDiscProps {
  label?: LabelExtensionProps;
}

export const BurstDisc = ({
  label,
  onChange,
  ...props
}: SymbolProps<BurstDiscProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.BurstDisc {...props} />
    </Labeled>
  );
};

export const BurstDiscPreview = (props: BurstDiscProps): ReactElement => (
  <Primitives.BurstDisc {...props} />
);

export interface CapProps extends Primitives.CapProps {
  label?: LabelExtensionProps;
}

export const Cap = ({
  label,
  onChange,
  ...props
}: SymbolProps<CapProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.Cap {...props} />
    </Labeled>
  );
};

export const CapPreview = (props: CapProps): ReactElement => (
  <Primitives.Cap {...props} />
);

export interface ManualValveProps extends Primitives.ManualValveProps {
  label?: LabelExtensionProps;
}

export const ManualValve = ({
  label,
  onChange,
  ...props
}: SymbolProps<ManualValveProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.ManualValve {...props} />
    </Labeled>
  );
};

export const ManualValvePreview = (props: ManualValveProps): ReactElement => (
  <Primitives.ManualValve {...props} />
);

export interface FilterProps extends Primitives.FilterProps {
  label?: LabelExtensionProps;
}

export const Filter = ({
  label,
  onChange,
  ...props
}: SymbolProps<FilterProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.Filter {...props} />
    </Labeled>
  );
};

export const FilterPreview = (props: FilterProps): ReactElement => (
  <Primitives.Filter {...props} />
);

export interface NeedleValveProps extends Primitives.NeedleValveProps {
  label?: LabelExtensionProps;
}

export const NeedleValve = ({
  label,
  onChange,
  ...props
}: SymbolProps<NeedleValveProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.NeedleValve {...props} />
    </Labeled>
  );
};

export const NeedleValvePreview = (props: NeedleValveProps): ReactElement => (
  <Primitives.NeedleValve {...props} />
);

export interface CheckValveProps extends Primitives.CheckValveProps {
  label?: LabelExtensionProps;
}

export const CheckValve = ({
  label,
  onChange,
  ...props
}: SymbolProps<CheckValveProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.CheckValve {...props} />
    </Labeled>
  );
};

export const CheckValvePreview = (props: CheckValveProps): ReactElement => (
  <Primitives.CheckValve {...props} />
);

export interface OrificeProps extends Primitives.OrificeProps {
  label?: LabelExtensionProps;
}

export const Orifice = ({
  label,
  onChange,
  ...props
}: SymbolProps<OrificeProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.Orifice {...props} />
    </Labeled>
  );
};

export const OrificePreview = (props: OrificeProps): ReactElement => (
  <Primitives.Orifice {...props} />
);

export interface AngledReliefValveProps extends Primitives.AngledReliefValveProps {
  label?: LabelExtensionProps;
}

export const AngledReliefValve = ({
  label,
  onChange,
  ...props
}: SymbolProps<AngledReliefValveProps>): ReactElement => {
  return (
    <Labeled {...label} onChange={onChange}>
      <Primitives.AngledReliefValve {...props} />
    </Labeled>
  );
};

export const AngledReliefValvePreview = (
  props: Primitives.AngledReliefValveProps,
): ReactElement => <Primitives.AngledReliefValve {...props} />;
