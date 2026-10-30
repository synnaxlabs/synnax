// Scratch page for visual options. Not for merge.

import { Breadcrumb } from "@synnaxlabs/lyra/breadcrumb";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { TimeStamp } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, type ReactNode, useState } from "react";

export const cls = (...c: (string | false | undefined | null)[]): string =>
  c.filter(Boolean).join(" ");

export const noop = (): void => {};

export type ThemeKey = "synnaxDark" | "synnaxLight";

const THEMES = {
  synnaxDark: { key: "synnaxDark" },
  synnaxLight: { key: "synnaxLight" },
} as const;

export const THEME_ORDER: ThemeKey[] = ["synnaxDark", "synnaxLight"];

interface ThemeBoxProps {
  theme: ThemeKey;
  children: ReactNode;
  className?: string;
}

export const ThemeBox = ({ theme, children, className }: ThemeBoxProps) => {
  const [el, setEl] = useState<HTMLDivElement | null>(null);
  return (
    <div ref={setEl} className={cls("opt-theme", className)}>
      <Theming.Provider theme={THEMES[theme]} el={el}>
        {el != null && children}
      </Theming.Provider>
    </div>
  );
};

interface OptionProps {
  letter: string;
  title: string;
  desc: string;
  className?: string;
  children: ReactNode | ((theme: ThemeKey) => ReactNode);
}

export const Option = ({ letter, title, desc, className, children }: OptionProps) => (
  <div className="opt-option">
    <div className="opt-option-label">
      <span className="opt-letter">{letter}</span>
      <span className="opt-title">{title}</span>
      <span className="opt-desc">{desc}</span>
    </div>
    <div className="opt-pair">
      {THEME_ORDER.map((t) => (
        <div className="opt-cell" key={t}>
          <div className="opt-cell-tag">{t === "synnaxDark" ? "Dark" : "Light"}</div>
          <ThemeBox theme={t}>
            <div className={cls("opt-body", className)}>
              {typeof children === "function" ? children(t) : children}
            </div>
          </ThemeBox>
        </div>
      ))}
    </div>
  </div>
);

interface SectionProps {
  id: string;
  n?: number;
  title: string;
  summary: string;
  className?: string;
  children: ReactNode;
}

export const Section = ({ id, n, title, summary, className, children }: SectionProps) => (
  <section className={cls("opt-section", className)} id={id}>
    <header className="opt-section-head">
      <h2>
        {n != null && <span className="opt-n">{n}</span>}
        {title}
      </h2>
      <p>{summary}</p>
    </header>
    <div className="opt-options">{children}</div>
  </section>
);

/** A small caption inside a themed cell, outside the component under review. */
export const Note = ({ children }: { children: ReactNode }) => (
  <span className="opt-note">{children}</span>
);

export const Row = ({
  label,
  children,
  align = "center",
}: {
  label?: string;
  children: ReactNode;
  align?: "center" | "start" | "end";
}) => (
  <div className="opt-row">
    {label != null && <Note>{label}</Note>}
    <Flex.Box x gap="medium" align={align} wrap className="opt-row-content">
      {children}
    </Flex.Box>
  </div>
);

export const StaticTooltip = ({
  children,
  className,
  style,
}: {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
}) => (
  <div role="tooltip" className={cls("pluto-tooltip", "opt-static", className)} style={style}>
    {typeof children === "string" ? (
      <Text.Text level="small" color={11}>
        {children}
      </Text.Text>
    ) : (
      children
    )}
  </div>
);

/** The Console wraps every context menu at the small level. */
export const StaticMenu = ({
  children,
  className,
  level = "small",
  gap = "small",
  style,
}: {
  children: ReactNode;
  className?: string;
  level?: "small" | "p";
  gap?: "small" | "medium";
  style?: CSSProperties;
}) => (
  <Flex.Box
    role="menu"
    gap="tiny"
    className={cls("pluto-menu-context", "pluto--bordered", "opt-static", className)}
    style={style}
  >
    <Menu.Menu level={level} gap={gap}>
      {children}
    </Menu.Menu>
  </Flex.Box>
);

export const StaticModal = ({
  children,
  className,
  style,
}: {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
}) => (
  <Flex.Box
    y
    pack
    empty
    align="stretch"
    background={0}
    rounded="small"
    bordered
    role="dialog"
    className={cls(
      "pluto-dialog__dialog",
      "pluto--modal",
      "pluto--visible",
      "pluto-modal",
      "opt-static",
      className,
    )}
    style={style}
  >
    {children}
  </Flex.Box>
);

/** The scrim a modal floats over, so the frosted panel reads as it does live. */
export const Scrim = ({ children }: { children: ReactNode }) => (
  <div className="opt-scrim">{children}</div>
);

export const SAVE: Triggers.Trigger = ["Control", "Enter"];

export const SaveHint = ({ action = "Save" }: { action?: string }) => (
  <Nav.Bar.Start gap="small">
    <Triggers.Text level="small" trigger={SAVE}>
      {action}
    </Triggers.Text>
  </Nav.Bar.Start>
);

/** A modal title variant: the subject at h4 in the main text color. */
export const ModalTitle = ({ children }: { children: string }) => (
  <Text.Text level="h4" weight={500} className="opt-modal-title">
    {children}
  </Text.Text>
);

/** A small breadcrumb above the title, with no bar under it. */
export const CrumbTitle = ({
  icon,
  crumbs,
  title,
}: {
  icon: Icon.ReactElement;
  crumbs: string[];
  title: string;
}) => (
  <Flex.Box x justify="between" align="start" className="opt-crumb-title">
    <Flex.Box y gap="tiny">
      <Breadcrumb.Breadcrumb gap="tiny" level="small">
        <Breadcrumb.Segment color={9}>{icon}</Breadcrumb.Segment>
        {crumbs.map((c) => (
          <Breadcrumb.Segment key={c} color={9} weight={400} level="small">
            {c}
          </Breadcrumb.Segment>
        ))}
      </Breadcrumb.Breadcrumb>
      <ModalTitle>{title}</ModalTitle>
    </Flex.Box>
    <Button.Button
      aria-label="Close"
      size="small"
      variant="text"
      textColor={9}
      tooltip="Close"
    >
      <Icon.Close />
    </Button.Button>
  </Flex.Box>
);

export { Modal };

export const now = TimeStamp.now();

export const CHANNELS = [
  "ox_tank_pressure_transducer_01",
  "ox_tank_pressure_transducer_02",
  "fuel_tank_pressure_transducer_01",
  "press_regulator_outlet_pressure",
  "engine_chamber_pressure_pt_101",
  "ox_main_fill_valve_cmd",
  "stand_alpha_time",
];

export const ESCAPE: Triggers.Trigger = ["Escape"];

export const Spacer = ({ size = 12 }: { size?: number }): ReactElement => (
  <div style={{ height: size, width: size, flexShrink: 0 }} />
);
