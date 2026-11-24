// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming as Base } from "@synnaxlabs/lyra/theming";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { Schematic, Theming } from "@synnaxlabs/pluto";
import { bounds, deep, dimensions, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { FileDrop } from "@/feature/schematic/symbol/edit/FileDrop";
import { HandleOverlay } from "@/feature/schematic/symbol/edit/Handles";
import {
  FLATTENED_ZOOM_TRIGGERS,
  ZOOM_TRIGGERS,
} from "@/feature/schematic/symbol/edit/triggers";
import { CSS } from "@/platform/css";

const path = Schematic.Symbol.formPath;

const ZOOM_OUT_TOOLTIP = (
  <Triggers.Text trigger={ZOOM_TRIGGERS.modes.out[0]} level="small">
    Zoom out
  </Triggers.Text>
);

const ZOOM_IN_TOOLTIP = (
  <Triggers.Text trigger={ZOOM_TRIGGERS.modes.in[0]} level="small">
    Zoom in
  </Triggers.Text>
);

const RESET_ZOOM_TOOLTIP = (
  <Triggers.Text trigger={ZOOM_TRIGGERS.modes.reset[0]} level="small">
    Reset zoom
  </Triggers.Text>
);

const MIN_ZOOM = 0.1;
const MAX_ZOOM = 5;
const ZOOM_STEP = 1.2;
const ZOOM_REPEAT = TimeSpan.milliseconds(70);

const VIEWPORT_PATH = deep.present(path.data.previewViewport);
const ZOOM_PATH = VIEWPORT_PATH.zoom;

const SELECTED_CLASS = CSS.BEM("schematic", "svg-region", "selected");

interface PreviewProps {
  /** selectedValue is the value of the state that the preview draws. */
  selectedValue: number;
  selectedHandle?: string;
  /** selectedPart is the part whose shapes the preview marks. */
  selectedPart?: string;
  /** onElementClick receives the markup with a name on the clicked shape. */
  onElementClick: (
    shape: Schematic.Node.Region.Named,
    click: Pick<MouseEvent, "detail" | "altKey">,
  ) => void;
  onContentsChange: (contents: string) => void;
  onHandleSelect: (handleKey: string) => void;
  onHandlePlace: (handleKey: string, position: { x: number; y: number }) => void;
}

export const Preview = ({
  selectedValue,
  selectedHandle,
  selectedPart,
  onElementClick,
  onContentsChange,
  onHandlePlace,
  onHandleSelect,
}: PreviewProps): ReactElement | null => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [drawingEl, setDrawingEl] = useState<SVGGElement | null>(null);
  const svgWrapperRef = useRef<HTMLDivElement>(null);
  // State, not a ref: the theme provider takes the element as a prop, and a ref filled
  // after the first render would never re-render it into place.
  const [themeContainer, setThemeContainer] = useState<HTMLDivElement | null>(null);
  const spec = Form.useFieldValue(path.data);
  const pan = Form.useField(VIEWPORT_PATH.position);
  const zoom = Form.useField(ZOOM_PATH);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const { dark: isDark, colors } = Base.use();
  // The toggle holds until the app theme changes.
  const [darkMode, setDarkMode] = useState({ app: isDark, dark: isDark });
  if (darkMode.app !== isDark) setDarkMode({ app: isDark, dark: isDark });
  const isDarkMode = darkMode.dark;

  const [svg, setSVG] = useState<SVGSVGElement | null>(null);
  const form = Form.useContext();

  const resetViewport = () => {
    zoom.onChange(1);
    pan.onChange({ x: 0, y: 0 });
  };

  // A held chord steps on an interval, so the step reads the live zoom from the form
  // rather than the one captured when the hold started.
  const stepZoom = (factor: number) => {
    const current = form.get<number>(ZOOM_PATH).value;
    form.set(
      ZOOM_PATH,
      bounds.clamp({ lower: MIN_ZOOM, upper: MAX_ZOOM }, current * factor),
    );
  };

  const handleZoomIn = () => stepZoom(ZOOM_STEP);
  const handleZoomOut = () => stepZoom(1 / ZOOM_STEP);
  const handleResetZoom = () => resetViewport();

  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault();
    stepZoom(e.deltaY > 0 ? 0.9 : 1.1);
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button === 1 || (e.button === 0 && e.shiftKey)) {
      e.preventDefault();
      setIsDragging(true);
      setDragStart({
        x: e.clientX - pan.value.x,
        y: e.clientY - pan.value.y,
      });
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging)
      pan.onChange({ x: e.clientX - dragStart.x, y: e.clientY - dragStart.y });
  };

  const handleMouseUp = () => setIsDragging(false);

  const repeat = useRef<ReturnType<typeof setInterval>>(null);
  const stopRepeat = () => {
    if (repeat.current == null) return;
    clearInterval(repeat.current);
    repeat.current = null;
  };
  useEffect(() => stopRepeat, [stopRepeat]);

  Triggers.use({
    triggers: FLATTENED_ZOOM_TRIGGERS,
    enabled: spec.drawing.markup.length > 0,
    callback: ({ triggers, stage }: Triggers.UseEvent) => {
      if (stage === "end") return stopRepeat();
      if (stage !== "start") return;
      const mode = Triggers.determineMode(ZOOM_TRIGGERS, triggers);
      if (mode === "reset") return handleResetZoom();
      if (mode !== "in" && mode !== "out") return;
      const step = mode === "in" ? handleZoomIn : handleZoomOut;
      step();
      // A held chord keeps zooming, the way a held arrow key keeps scrolling.
      stopRepeat();
      repeat.current = setInterval(step, ZOOM_REPEAT.milliseconds);
    },
  });

  const onMount = (drawing: SVGGElement) => {
    setDrawingEl(drawing);
    const addInteractivity = (el: Element) => {
      if (!(el instanceof SVGElement) || el.tagName === "svg") return;
      el.classList.add(CSS.BEM("schematic", "svg-region", "hoverable"));
      // The CSS filter of a hover or a selection replaces the filter of the drawing.
      const own = el.style.filter || el.getAttribute("filter");
      el.style.removeProperty("filter");
      if (own != null && own !== "" && own !== "none")
        el.style.setProperty("--console-schematic-own-filter", own);
      if (
        el.tagName === "line" ||
        el.tagName === "polyline" ||
        (el.tagName === "path" && el.getAttribute("fill") === "none")
      ) {
        const originalStrokeWidth = el.getAttribute("stroke-width") || "1";
        el.setAttribute("data-original-stroke-width", originalStrokeWidth);
        const currentWidth = parseFloat(originalStrokeWidth);
        if (currentWidth < 5) {
          el.setAttribute("stroke-width", "5");
          el.style.strokeOpacity = "0.3";
        }
      }

      el.addEventListener("click", (e) => {
        // The groups around the shape have their own listeners.
        e.stopPropagation();
        // Shift+drag pans the preview.
        if (e.shiftKey) return;
        const markup = form.get(path.data.drawing.markup).value;
        onElementClick(Schematic.Node.Region.name(markup, drawing, el), e);
      });

      // The spans of a text are part of its shape, so their clicks reach the text.
      if (el.tagName !== "text") Array.from(el.children).forEach(addInteractivity);
    };
    Array.from(drawing.children).forEach(addInteractivity);
  };

  const selected =
    spec.parts.find((p) => p.key === selectedPart)?.selectors.join(", ") ?? "";
  useEffect(() => {
    if (drawingEl == null || selected === "") return;
    const shapes = Array.from(drawingEl.querySelectorAll(selected));
    for (const el of shapes) el.classList.add(SELECTED_CLASS);
    return () => {
      for (const el of shapes) el.classList.remove(SELECTED_CLASS);
    };
  }, [drawingEl, selected]);

  // A schematic node gives the node color, so the preview draws a sample of it.
  const paints = Schematic.Node.Custom.usePaints(
    spec,
    selectedValue,
    undefined,
    colors.primary.z,
  );
  const setDrawing = Schematic.Node.Custom.useRender({ spec, paints, onMount });
  const dims = Schematic.Node.Region.measure(spec.drawing.markup);

  const handleContentsChange = (contents: string, filename?: string) => {
    const processedSVG = Schematic.Node.Region.normalizeSVG(contents);
    if (containerRef.current == null) return;
    onContentsChange(processedSVG);

    if (filename != null && form.get(path.name).value === "")
      form.set(path.name, filename);

    const parser = new DOMParser();
    const svgDoc = parser.parseFromString(processedSVG, "image/svg+xml");
    const svgElement = svgDoc.documentElement as unknown as SVGElement;
    // The drawing keeps its own colors, so its parts need no looks.
    const parts = Schematic.Node.Region.extract(svgElement).map(
      ({ key, name, selectors }) => ({ key, name, selectors }),
    );
    form.set(path.data.parts, parts);
    form.set(path.data.looks, []);
  };

  const fileDropEnabled = spec.drawing.markup.length === 0;

  const svgWrapperStyle = {
    transform: `translate(${pan.value.x}px, ${pan.value.y}px) scale(${zoom.value})`,
    transition: isDragging ? "none" : "transform 0.2s ease-out",
    cursor: isDragging ? "grabbing" : "default",
  };

  return (
    <FileDrop
      onContentsChange={handleContentsChange}
      grow={1}
      enabled={fileDropEnabled}
    >
      <Theming.Provider
        el={themeContainer}
        theme={Base.SYNNAX_THEMES[isDarkMode ? "synnaxDark" : "synnaxLight"]}
      >
        <Flex.Box
          ref={setThemeContainer}
          className={CSS.cls(
            CSS.B("schematic-preview-theme-container"),
            fileDropEnabled && CSS.M("hidden"),
          )}
          onWheel={handleWheel}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
          background={0}
          rounded="small"
        >
          {spec.drawing.markup.length > 0 && (
            <Flex.Box x className={CSS.B("schematic-preview-controls")}>
              <Text.Text level="small" color={9}>
                {Math.round(zoom.value * 100)}%
              </Text.Text>
              <Button.Button
                variant="text"
                size="small"
                onClick={() => setDarkMode({ app: isDark, dark: !isDarkMode })}
              >
                {isDarkMode ? <Icon.DarkMode /> : <Icon.LightMode />}
              </Button.Button>
              <Flex.Box pack x>
                <Button.Button
                  onClick={handleZoomOut}
                  size="small"
                  tooltip={ZOOM_OUT_TOOLTIP}
                >
                  <Icon.Subtract />
                </Button.Button>
                <Button.Button
                  onClick={handleZoomIn}
                  size="small"
                  tooltip={ZOOM_IN_TOOLTIP}
                >
                  <Icon.Add />
                </Button.Button>
                <Button.Button
                  onClick={handleResetZoom}
                  size="small"
                  tooltip={RESET_ZOOM_TOOLTIP}
                >
                  <Icon.Expand />
                </Button.Button>
              </Flex.Box>
            </Flex.Box>
          )}
          <Flex.Box
            center
            ref={svgWrapperRef}
            className={CSS.B("schematic-preview-svg-wrapper")}
            style={svgWrapperStyle}
          >
            <div className={CSS.B("schematic-preview-svg-wrapper-inner")}>
              <HandleOverlay
                handles={spec.handles}
                selectedHandle={selectedHandle}
                svg={svg}
                onSelect={onHandleSelect}
                onDrag={onHandlePlace}
              />
              <div ref={containerRef} className={CSS.B("preview")}>
                {dims != null && (
                  <svg
                    ref={setSVG}
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox={dimensions.svgViewBox(dims)}
                    {...dimensions.scale(
                      dims,
                      Schematic.Node.Primitive.resolveScale(spec.scale),
                    )}
                  >
                    <g ref={setDrawing} />
                  </svg>
                )}
              </div>
            </div>
          </Flex.Box>
        </Flex.Box>
      </Theming.Provider>
    </FileDrop>
  );
};
