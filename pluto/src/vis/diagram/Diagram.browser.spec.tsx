// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/diagram/Diagram.css";
import "@xyflow/react/dist/base.css";

import { Component } from "@synnaxlabs/lyra/component";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { TimeSpan } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { useReactFlow } from "@xyflow/react";
import { type CSSProperties, type ReactElement, type ReactNode, useState } from "react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { render } from "@/testutil";
import { Viewport } from "@/viewport";
import { Diagram } from "@/vis/diagram";
import { diagram } from "@/vis/diagram/aether";

// The nodrag child stands in for a connection handle: React Flow's drag filter rejects
// it, so its mousedown reaches the window like a click on the canvas would.
const NodeView = ({ nodeKey }: Diagram.NodeProps): ReactElement => (
  <div>
    {nodeKey}
    <span className="nodrag" data-testid={`handle-${nodeKey}`} />
  </div>
);

const Base = Diagram.create({ node: Component.renderProp(NodeView) });

const NODES: Diagram.Node[] = [
  { key: "a", position: { x: 0, y: 0 } },
  { key: "b", position: { x: 200, y: 0 } },
];
const VIEWPORT: Diagram.Viewport = { position: { x: 0, y: 0 }, zoom: 1 };
const noop = () => {};

interface HarnessProps {
  children?: ReactNode;
  themeKey?: string;
  onSelectionChange: (selected: string[]) => void;
  onViewportChange: (viewport: Diagram.Viewport) => void;
  editable?: boolean;
  onPaste?: () => void;
  viewportMode?: Viewport.Mode;
}

// The diagram fills its parent, so the parent gives it a size.
const SIZE: CSSProperties = { width: 1000, height: 500 };

// The diagram is controlled, so the harness feeds each selection back as the next
// `selected` prop the way a caller would.
const Harness = ({
  themeKey = "synnaxLight",
  onSelectionChange,
  onViewportChange,
  children,
  editable = true,
  onPaste,
  viewportMode = "zoom",
}: HarnessProps): ReactElement => {
  const [selected, setSelected] = useState<string[]>([]);
  const handleSelectionChange = (next: string[]): void => {
    setSelected(next);
    onSelectionChange(next);
  };
  return (
    <Theming.Provider theme={{ key: themeKey }} applyCSSVars={false}>
      <Triggers.Provider>
        <Tooltip.Config delay={TimeSpan.milliseconds(1)}>
          <div style={SIZE}>
            <Base
              nodes={NODES}
              edges={[]}
              onNodesChange={noop}
              onEdgesChange={noop}
              selected={selected}
              onSelectionChange={handleSelectionChange}
              editable={editable}
              onEditableChange={noop}
              onPaste={onPaste}
              viewport={VIEWPORT}
              onViewportChange={onViewportChange}
              fitViewOnResize={false}
              setFitViewOnResize={noop}
              viewportMode={viewportMode}
              onViewportModeChange={noop}
              triggers={Viewport.DEFAULT_TRIGGERS[viewportMode]}
              visible
            >
              {children}
            </Base>
          </div>
        </Tooltip.Config>
      </Triggers.Provider>
    </Theming.Provider>
  );
};

const renderDiagram = async (
  children?: ReactNode,
  props: Pick<HarnessProps, "editable" | "onPaste" | "themeKey" | "viewportMode"> = {},
) => {
  const onSelectionChange = vi.fn<(selected: string[]) => void>();
  const onViewportChange = vi.fn<(viewport: Diagram.Viewport) => void>();
  render(
    <Harness
      onSelectionChange={onSelectionChange}
      onViewportChange={onViewportChange}
      {...props}
    >
      {children}
    </Harness>,
    { registry: diagram.REGISTRY, render: true },
  );
  await waitFor(() => expect(node("b")).not.toBeNull());
  return { onSelectionChange, onViewportChange };
};

const node = (key: string): HTMLElement =>
  document.querySelector(`.react-flow__node[data-id="${key}"]`) as HTMLElement;

const handle = (key: string): HTMLElement =>
  document.querySelector(`[data-testid="handle-${key}"]`) as HTMLElement;

const pane = (): HTMLElement =>
  document.querySelector(".react-flow__pane") as HTMLElement;

describe("Diagram", () => {
  // Canvas.useRegion measures against the lower2d canvas and bails without it.
  const canvas = document.createElement("div");

  beforeAll(() => {
    canvas.className = "pluto-canvas--lower2d";
    document.body.appendChild(canvas);
  });

  afterAll(() => {
    canvas.remove();
    vi.unstubAllGlobals();
  });

  describe("controls", () => {
    it("names the fit view button for assistive technology", async () => {
      await renderDiagram(<Diagram.Controls.FitView />);
      expect(
        screen.getByRole("button", { name: "Fit view to contents" }),
      ).toBeDefined();
    });

    it.each([
      { mode: "pan", button: 0, keys: 1 },
      { mode: "pan", button: 1, keys: 2 },
      { mode: "select", button: 0, keys: 2 },
      { mode: "select", button: 1, keys: 1 },
    ] as const)(
      "hints the gesture that pans or selects in $mode mode (button $button)",
      async ({ mode, button, keys }) => {
        await renderDiagram(<Diagram.Controls.SelectViewportMode />, {
          viewportMode: mode,
        });
        const buttons = document.querySelectorAll(".lyra-select-btns button");
        fireEvent.pointerEnter(buttons[button]);
        await waitFor(() => {
          const tip = document.querySelector(".lyra-tooltip");
          if (tip == null) throw new Error("tooltip did not open");
          expect(tip.querySelectorAll(".lyra-triggers__keys > *")).toHaveLength(keys);
        });
      },
    );
  });

  describe("selection", () => {
    it("replaces the selection on a plain click", async () => {
      const { onSelectionChange } = await renderDiagram();
      fireEvent.click(node("a"));
      expect(onSelectionChange).toHaveBeenLastCalledWith(["a"]);
      fireEvent.click(node("b"));
      expect(onSelectionChange).toHaveBeenLastCalledWith(["b"]);
    });

    it.each(["Meta", "Control", "Shift"])(
      "toggles the clicked node in the selection while %s is held",
      async (key) => {
        const { onSelectionChange } = await renderDiagram();
        fireEvent.click(node("a"));
        fireEvent.keyDown(window, { key, code: `${key}Left` });
        fireEvent.click(node("b"));
        expect(onSelectionChange).toHaveBeenLastCalledWith(["a", "b"]);
        fireEvent.click(node("a"));
        expect(onSelectionChange).toHaveBeenLastCalledWith(["b"]);
        fireEvent.keyUp(window, { key, code: `${key}Left` });
        fireEvent.click(node("a"));
        expect(onSelectionChange).toHaveBeenLastCalledWith(["a"]);
      },
    );
  });

  describe("color mode", () => {
    const flowRoot = (): HTMLElement =>
      document.querySelector(".react-flow") as HTMLElement;

    const preferDark = (dark: boolean): void => {
      vi.stubGlobal("matchMedia", (query: string): Partial<MediaQueryList> => ({
        matches: dark && query === "(prefers-color-scheme: dark)",
        addEventListener: noop,
        removeEventListener: noop,
      }));
    };

    it("follows a light theme when the OS prefers dark", async () => {
      preferDark(true);
      await renderDiagram(undefined, { themeKey: "synnaxLight" });
      expect(flowRoot().classList.contains("light")).toBe(true);
      expect(flowRoot().classList.contains("dark")).toBe(false);
    });

    it("follows a dark theme when the OS prefers light", async () => {
      preferDark(false);
      await renderDiagram(undefined, { themeKey: "synnaxDark" });
      expect(flowRoot().classList.contains("dark")).toBe(true);
      expect(flowRoot().classList.contains("light")).toBe(false);
    });
  });

  describe("zoom reset", () => {
    // Moves the view away from the fit, so a reset has something to change.
    const Pan = (): ReactElement => {
      const { setViewport } = useReactFlow();
      return (
        <button onClick={() => void setViewport({ x: 40, y: 40, zoom: 0.5 })}>
          Pan
        </button>
      );
    };

    // The diagram fits its view once React Flow measures the nodes. The spec waits for
    // that fit, then pans away from it and holds Control over the canvas.
    const panAndHoldControl = async (): Promise<
      ReturnType<typeof vi.fn<(viewport: Diagram.Viewport) => void>>
    > => {
      const { onViewportChange } = await renderDiagram(<Pan />);
      // The mount reports the initial viewport, and the fit reports a new one.
      await waitFor(() =>
        expect(onViewportChange).toHaveBeenCalledWith(
          expect.not.objectContaining({ position: VIEWPORT.position }),
        ),
      );
      onViewportChange.mockClear();
      fireEvent.click(screen.getByRole("button", { name: "Pan" }));
      await waitFor(() => expect(onViewportChange).toHaveBeenCalled());
      const { left, top } = pane().getBoundingClientRect();
      fireEvent.mouseMove(window, { clientX: left + 10, clientY: top + 10 });
      fireEvent.keyDown(window, { key: "Control", code: "ControlLeft" });
      onViewportChange.mockClear();
      return onViewportChange;
    };

    it("resets the view on a Control-click of the canvas", async () => {
      const onViewportChange = await panAndHoldControl();
      fireEvent.mouseDown(pane(), { button: 0 });
      await waitFor(() => expect(onViewportChange).toHaveBeenCalled());
    });

    it("leaves the view alone on a Control-click of a node", async () => {
      const onViewportChange = await panAndHoldControl();
      fireEvent.mouseDown(handle("a"), { button: 0 });
      await act(async () => {});
      expect(onViewportChange).not.toHaveBeenCalled();
    });
  });

  describe("clipboard", () => {
    it("does not paste while the diagram is not editable", async () => {
      const onPaste = vi.fn();
      await renderDiagram(undefined, { editable: false, onPaste });
      fireEvent.paste(document.querySelector(".pluto-diagram__container") as Element);
      expect(onPaste).not.toHaveBeenCalled();
    });
  });
});

const renderStyled = async (colorMode: "light" | "dark"): Promise<HTMLElement> => {
  const { container } = await renderThemed(
    <div className={CSS.BE("diagram", "container")}>
      <div className={CSS.cls("react-flow", colorMode, CSS.B("diagram"))} />
    </div>,
  );
  const diagram = container.querySelector<HTMLElement>(".react-flow");
  if (diagram == null) throw new Error("expected a diagram element");
  return diagram;
};

describe("Diagram style", () => {
  it.each(["light", "dark"] as const)(
    "should show the theme through the background in the %s color mode",
    async (colorMode) => {
      const diagram = await renderStyled(colorMode);
      expect(getComputedStyle(diagram).backgroundColor).toBe("rgba(0, 0, 0, 0)");
    },
  );
});
