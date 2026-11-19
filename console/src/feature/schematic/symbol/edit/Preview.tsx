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

interface PreviewProps {
  /** selectedValue is the value of the state that the preview draws. */
  selectedValue: number;
  selectedHandle?: string;
  onElementClick: (selector: string) => void;
  onContentsChange: (contents: string) => void;
  onHandleSelect: (handleKey: string) => void;
  onHandlePlace: (handleKey: string, position: { x: number; y: number }) => void;
}

export const Preview = ({
  selectedValue,
  selectedHandle,
  onElementClick,
  onContentsChange,
  onHandlePlace,
  onHandleSelect,
}: PreviewProps): ReactElement | null => {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgWrapperRef = useRef<HTMLDivElement>(null);
  // State, not a ref: the theme provider takes the element as a prop, and a ref filled
  // after the first render would never re-render it into place.
  const [themeContainer, setThemeContainer] = useState<HTMLDivElement | null>(null);
  const spec = Form.useFieldValue(path.data);
  const pan = Form.useField(VIEWPORT_PATH.position);
  const zoom = Form.useField(ZOOM_PATH);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const isDark = Base.use().dark;
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
    const addInteractivity = (el: Element) => {
      if (!(el instanceof SVGElement) || el.tagName === "svg") return;
      el.classList.add(CSS.BEM("schematic", "svg-region", "hoverable"));
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

      el.addEventListener("mouseenter", () => {
        if (el.tagName === "line" || el.tagName === "polyline")
          el.style.filter = "drop-shadow(0 0 3px rgba(99, 102, 241, 0.8))";
        else {
          const fillAttr = el.getAttribute("fill");
          const hasVisibleFill =
            fillAttr && fillAttr !== "none" && fillAttr !== "transparent";
          if (hasVisibleFill) el.style.filter = "brightness(1.3)";
          else {
            el.style.fill = "rgba(99, 102, 241, 0.2)";
            el.style.fillOpacity = "1";
          }
        }
        el.classList.add(CSS.BEM("schematic", "svg-region", "hover"));
      });

      el.addEventListener("mouseleave", () => {
        el.style.filter = "";
        el.style.fill = "";
        el.style.fillOpacity = "";
        el.classList.remove(CSS.BEM("schematic", "svg-region", "hover"));
      });

      el.addEventListener("click", () => {
        const markup = form.get(path.data.drawing.markup).value;
        const named = Schematic.Node.Region.name(markup, drawing, el);
        if (named.markup !== markup) form.set(path.data.drawing.markup, named.markup);
        onElementClick(named.selector);
      });

      Array.from(el.children).forEach(addInteractivity);
    };
    Array.from(drawing.children).forEach(addInteractivity);
  };

  const paints = Schematic.Node.Custom.usePaints(spec, selectedValue);
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
