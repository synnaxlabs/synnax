import "@/schematic/symbols/Create.css";

import { Button, Color, Flex, Haul, Icon, Nav, Status, Text } from "@synnaxlabs/pluto";
import { color, uuid } from "@synnaxlabs/x";
import { open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";
import React, { type ReactElement, useEffect, useRef, useState } from "react";

import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const CREATE_LAYOUT_TYPE = "schematic_edit_symbol";

export interface CreateLayoutArgs {}

export const CREATE_LAYOUT: Layout.BaseState = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  location: "modal",
  name: "Schematic.Create Symbol",
  icon: "Schematic",
  window: {
    resizable: false,
    size: { width: 600, height: 500 },
    navTop: true,
    showTitle: true,
  },
};

export const createCreateLayout = (
  initial: CreateLayoutArgs = {},
): Layout.BaseState<CreateLayoutArgs> => ({
  ...CREATE_LAYOUT,
  args: initial,
});

const canDrop: Haul.CanDrop = ({ items }) =>
  items.some((item) => item.type === Haul.FILE_TYPE) && items.length === 1;

interface StateConfig {
  id: string;
  name: string;
  regions: RegionConfig[];
  color: string;
}

interface BaseColorConfig {
  regions: RegionConfig[];
  color: string;
}

interface RegionConfig {
  id: string;
  selector: string;
  name?: string;
  fillColor?: string;
  strokeColor?: string;
  useCustomColors?: boolean;
}

export const Create: Layout.Renderer = (): ReactElement => {
  const [svgContent, setSvgContent] = useState<string>("");
  const [svgPath, setSvgPath] = useState<string>("");
  const [draggingOver, setDraggingOver] = useState(false);
  const [hoveredElement, setHoveredElement] = useState<SVGElement | null>(null);
  const [activeRegion, setActiveRegion] = useState<string | null>(null); // ID of currently selected region
  const [currentMode, setCurrentMode] = useState<string>("active"); // Can be state ID or "base"
  const [states, setStates] = useState<StateConfig[]>([
    { id: "active", name: "Active", regions: [], color: "#10b981" },
    { id: "inactive", name: "Inactive", regions: [], color: "#6b7280" },
  ]);
  const [baseColorState, setBaseColorState] = useState<BaseColorConfig>({
    regions: [],
    color: "#f59e0b",
  }); // Base amber
  const svgContainerRef = useRef<HTMLDivElement>(null);

  // Refs to avoid stale closures in event handlers
  const currentModeRef = useRef(currentMode);
  const statesRef = useRef(states);
  const baseColorStateRef = useRef(baseColorState);

  // Keep refs in sync
  useEffect(() => {
    currentModeRef.current = currentMode;
  }, [currentMode]);

  useEffect(() => {
    statesRef.current = states;
  }, [states]);

  useEffect(() => {
    baseColorStateRef.current = baseColorState;
  }, [baseColorState]);
  const handleError = Status.useErrorHandler();

  const handleFileSelect = () =>
    handleError(async () => {
      const path = await open({
        directory: false,
        filters: [{ name: "SVG Files", extensions: ["svg"] }],
      });
      if (path == null) return;
      const contents = await readFile(path);
      if (contents == null) return;
      const svgText = new TextDecoder().decode(contents);
      setSvgContent(svgText);
      setSvgPath(path);
      setStates([
        { id: "active", name: "Active", regions: [], color: "#10b981" },
        { id: "inactive", name: "Inactive", regions: [], color: "#6b7280" },
      ]);
      setBaseColorState({ regions: [], color: "#f59e0b" });
      setHoveredElement(null);
      setActiveRegion(null);
    }, "Failed to load SVG file");

  const handleFileDrop = ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
    if (event == null) return items;
    event.preventDefault();
    setDraggingOver(false);
    if (event.dataTransfer.files.length === 0) return items;

    const file = event.dataTransfer.files[0];
    if (!file.name.toLowerCase().endsWith(".svg")) {
      handleError(() => {
        throw new Error("Please select an SVG file");
      }, "Invalid file type");
      return items;
    }

    handleError(async () => {
      const text = await file.text();
      setSvgContent(text);
      setSvgPath(file.name);
      setStates([
        { id: "active", name: "Active", regions: [], color: "#10b981" },
        { id: "inactive", name: "Inactive", regions: [], color: "#6b7280" },
      ]);
      setBaseColorState({ regions: [], color: "#f59e0b" });
      setHoveredElement(null);
      setActiveRegion(null);
    }, "Failed to load dropped SVG file");
    return items;
  };

  const dropProps = Haul.useDrop({
    type: Haul.FILE_TYPE,
    onDrop: handleFileDrop,
    canDrop,
    onDragOver: () => setDraggingOver(true),
  });

  const injectSVG = (svgString: string) => {
    if (!svgContainerRef.current) return;

    // Parse SVG string
    const parser = new DOMParser();
    const svgDoc = parser.parseFromString(svgString, "image/svg+xml");
    const svgElement = svgDoc.documentElement;

    // Clear container
    svgContainerRef.current.innerHTML = "";

    // Add event listeners to all SVG elements
    const addInteractivity = (element: Element) => {
      if (element instanceof SVGElement && element.tagName !== "svg") {
        element.classList.add(CSS.BEM("schematic", "svg-region", "hoverable"));

        element.addEventListener("mouseenter", () => {
          setHoveredElement(element);
          
          // Special handling for stroke-only elements (lines, polylines)
          if (element.tagName === "line" || element.tagName === "polyline") {
            // Use glow effect for pure stroke elements
            element.style.filter = "drop-shadow(0 0 3px rgba(99, 102, 241, 0.8))";
          } else {
            // Check if element has a visible fill color
            const fillAttr = element.getAttribute("fill");
            const hasVisibleFill = fillAttr && fillAttr !== "none" && fillAttr !== "transparent";
            
            if (hasVisibleFill) {
              // Apply brightness filter to existing fill
              element.style.filter = "brightness(1.3)";
            } else {
              // Apply light blue overlay for transparent/none fills
              element.style.fill = "rgba(99, 102, 241, 0.2)";
              element.style.fillOpacity = "1";
            }
          }
          
          element.classList.add(CSS.BEM("schematic", "svg-region", "hover"));
        });

        element.addEventListener("mouseleave", () => {
          setHoveredElement(null);
          
          // Clear hover effects
          element.style.filter = "";
          element.style.fill = "";
          element.style.fillOpacity = "";
          
          element.classList.remove(CSS.BEM("schematic", "svg-region", "hover"));
        });

        element.addEventListener("click", (e) => {
          e.stopPropagation();

          // Generate selector at click time
          let selector: string;
          if (element.id) selector = `#${element.id}`;
          else {
            // Check if element already has data-region-id
            const existingDataId = element.getAttribute("data-region-id");
            if (existingDataId) selector = `[data-region-id="${existingDataId}"]`;
            else {
              // Generate new data attribute for selection
              const regionId = uuid.create();
              const dataId = `region-${regionId}`;
              element.setAttribute("data-region-id", dataId);
              selector = `[data-region-id="${dataId}"]`;
            }
          }

          handleRegionClick(element, selector);
        });
      }

      // Recursively add to children
      Array.from(element.children).forEach(addInteractivity);
    };

    addInteractivity(svgElement);
    svgContainerRef.current.appendChild(svgElement);
  };

  const handleRegionClick = (element: SVGElement, selector: string) => {
    // Use refs to get current values and avoid stale closures
    const currentModeValue = currentModeRef.current;
    const currentStates = statesRef.current;
    const currentBaseState = baseColorStateRef.current;

    console.log("Clicking element in mode:", currentModeValue);

    // We'll use the region ID from the region object itself for consistency

    if (currentModeValue === "base") {
      // Handle base colors
      const existingRegion = currentBaseState.regions.find(
        (r) => r.selector === selector,
      );

      if (existingRegion) {
        console.log("Removing region from base colors");
        setBaseColorState((prev) => ({
          ...prev,
          regions: prev.regions.filter((r) => r.id !== existingRegion.id),
        }));
        setActiveRegion(null);
      } else {
        console.log("Adding region to base colors");
        const regionId = uuid.create();
        const newRegion: RegionConfig = {
          id: regionId,
          selector,
          name: element.tagName.toLowerCase(),
          useCustomColors: false,
        };
        setBaseColorState((prev) => ({
          ...prev,
          regions: [...prev.regions, newRegion],
        }));
        setActiveRegion(regionId);
      }
    } else {
      // Handle dynamic states
      const targetStateIndex = currentStates.findIndex(
        (s) => s.id === currentModeValue,
      );
      if (targetStateIndex === -1) return;

      const targetState = currentStates[targetStateIndex];
      const existingRegion = targetState.regions.find((r) => r.selector === selector);

      if (existingRegion) {
        console.log("Removing region from state:", currentModeValue);
        setStates((prev) =>
          prev.map((state, index) =>
            index === targetStateIndex
              ? {
                  ...state,
                  regions: state.regions.filter((r) => r.id !== existingRegion.id),
                }
              : state,
          ),
        );
        setActiveRegion(null);
      } else {
        console.log("Adding region to state:", currentModeValue);
        const regionId = uuid.create();
        const newRegion: RegionConfig = {
          id: regionId,
          selector,
          name: element.tagName.toLowerCase(),
          useCustomColors: false,
        };
        setStates((prev) =>
          prev.map((state, index) =>
            index === targetStateIndex
              ? {
                  ...state,
                  regions: [...state.regions, newRegion],
                }
              : state,
          ),
        );
        setActiveRegion(regionId);
      }
    }
  };

  // Apply live preview styling to main SVG based on current mode
  const applyLivePreview = () => {
    if (!svgContainerRef.current) return;

    console.log("Applying live preview for mode:", currentMode);

    // Clear ALL visual styling from ALL elements (not just data-region-id ones)
    const allElements = svgContainerRef.current.querySelectorAll("*");
    allElements.forEach((element) => {
      if (element instanceof SVGElement) {
        element.removeAttribute("fill");
        element.removeAttribute("fill-opacity");
        element.style.stroke = "";
        element.style.strokeWidth = "";
        element.style.filter = "";
        element.classList.remove(CSS.BEM("schematic", "svg-region", "selected"));
        element.classList.remove(CSS.BEM("schematic", "svg-region", "hover"));
        element.classList.remove(CSS.BEM("schematic", "svg-region", "active"));
      }
    });

    // Apply styling based on current mode only - this is a preview of what you're configuring
    console.log("Applying preview for current mode:", currentMode);

    // First apply base color overrides (lowest priority)
    baseColorState.regions.forEach((region) => {
      const element = svgContainerRef.current?.querySelector(region.selector);
      if (element && element instanceof SVGElement)
        if (region.useCustomColors) {
          if (region.fillColor) {
            element.setAttribute("fill", region.fillColor);
            element.setAttribute("fill-opacity", "1");
          }
          if (region.strokeColor) {
            element.style.stroke = region.strokeColor;
            element.style.strokeWidth = "2px";
          }
        } else {
          element.setAttribute("fill", baseColorState.color);
          element.setAttribute("fill-opacity", "1");
        }
      // Don't add selected class here - only for visual grouping
    });

    // Then apply current mode's colors (highest priority for preview)
    if (currentMode !== "base") {
      const currentState = states.find((s) => s.id === currentMode);
      if (currentState)
        currentState.regions.forEach((region) => {
          const element = svgContainerRef.current?.querySelector(region.selector);
          if (element && element instanceof SVGElement)
            if (region.useCustomColors) {
              if (region.fillColor) {
                element.setAttribute("fill", region.fillColor);
                element.setAttribute("fill-opacity", "1");
              }
              if (region.strokeColor) {
                element.style.stroke = region.strokeColor;
                element.style.strokeWidth = "2px";
              }
            } else {
              element.setAttribute("fill", currentState.color);
              element.setAttribute("fill-opacity", "1");
            }
          // Don't add selected class here - only for visual grouping
        });
    }
    // Note: If currentMode === "base", only base colors are applied

    // Highlight the single active region
    if (activeRegion) {
      // Find the element using region ID - it could be stored as data attribute or element ID
      let element: SVGElement | null = null;

      // First try to find by data-region-id
      element = svgContainerRef.current?.querySelector(
        `[data-region-id="region-${activeRegion}"]`,
      ) as SVGElement;

      // If not found and activeRegion looks like an element ID, try finding by ID
      if (!element && activeRegion && !activeRegion.includes("-"))
        element = svgContainerRef.current?.querySelector(
          `#${activeRegion}`,
        ) as SVGElement;

      // If still not found, try to find the region by matching against all regions
      if (!element) {
        const allRegions = [
          ...baseColorState.regions,
          ...states.flatMap((s) => s.regions),
        ];
        const targetRegion = allRegions.find((r) => r.id === activeRegion);
        if (targetRegion)
          element = svgContainerRef.current?.querySelector(
            targetRegion.selector,
          ) as SVGElement;
      }

      if (element && element instanceof SVGElement)
        element.classList.add(CSS.BEM("schematic", "svg-region", "active"));
    }
  };

  // Inject SVG when content changes
  useEffect(() => {
    if (svgContent) {
      injectSVG(svgContent);
      applyLivePreview();
    }
  }, [svgContent]);

  // Update live preview when mode, states, or active region changes
  useEffect(() => {
    applyLivePreview();
  }, [currentMode, states, baseColorState, activeRegion]);

  return (
    <Flex.Box y full className={CSS.BE("schematic", "create", "symbol")}>
      <Flex.Box grow className={CSS.BE("schematic", "create", "symbol", "content")}>
        {svgContent === "" ? (
          <Flex.Box
            grow
            align="center"
            justify="center"
            bordered
            background={draggingOver ? 2 : 1}
            onDragLeave={() => setDraggingOver(false)}
            rounded={2}
            onClick={handleFileSelect}
            {...dropProps}
            borderColor={6}
          >
            <Flex.Box y align="center" style={{ gap: "1rem" }}>
              <Text.Text level="h1" color={7}>
                <Icon.Import />
              </Text.Text>
              <Text.Text level="p">
                Click to select an SVG file or drag and drop it here
              </Text.Text>
            </Flex.Box>
          </Flex.Box>
        ) : (
          <Flex.Box x grow gap={2}>
            <Flex.Box y grow gap={1}>
              <Flex.Box x justify="between" align="center">
                <Text.Text level="p">
                  Configuring:{" "}
                  <strong>
                    {currentMode === "base"
                      ? "Base Colors"
                      : states.find((s) => s.id === currentMode)?.name || "Unknown"}
                  </strong>{" "}
                  state
                </Text.Text>
                <Button.Button
                  variant="outlined"
                  size="small"
                  onClick={() => {
                    setSvgContent("");
                    setSvgPath("");
                    setStates([
                      { id: "active", name: "Active", regions: [], color: "#10b981" },
                      {
                        id: "inactive",
                        name: "Inactive",
                        regions: [],
                        color: "#6b7280",
                      },
                    ]);
                    setBaseColorState({ regions: [], color: "#f59e0b" });
                    setHoveredElement(null);
                    setActiveRegion(null);
                  }}
                >
                  <Icon.Close />
                  Clear
                </Button.Button>
              </Flex.Box>

              <Flex.Box
                grow
                align="center"
                justify="center"
                className={CSS.BE("schematic", "svg-container")}
                style={{
                  border: "1px solid var(--pluto-gray-l5)",
                  borderRadius: "8px",
                  backgroundColor: "var(--pluto-gray-l2)",
                  padding: "1rem",
                  overflow: "hidden",
                }}
              >
                <div
                  ref={svgContainerRef}
                  className={CSS.BE("schematic", "svg-content")}
                  style={{
                    maxWidth: "100%",
                    maxHeight: "100%",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                  }}
                />
              </Flex.Box>

              {svgPath && (
                <Text.Text level="small" style={{ color: "var(--pluto-gray-l8)" }}>
                  File: {svgPath.split("/").pop() || svgPath.split("\\").pop()}
                </Text.Text>
              )}
            </Flex.Box>

            {/* State Configuration Panel */}
            <Flex.Box y style={{ minWidth: "250px", maxWidth: "300px" }}>
              {/* State Mode Toggle */}
              <Flex.Box y style={{ marginBottom: "1rem" }}>
                <Text.Text level="p" style={{ marginBottom: "0.5rem" }}>
                  Configure States
                </Text.Text>
                <Flex.Box y style={{ gap: "0.25rem" }}>
                  {/* Render dynamic states */}
                  <Flex.Box x style={{ gap: "0.5rem", flexWrap: "wrap" }}>
                    {states.map((state) => (
                      <Flex.Box
                        key={state.id}
                        x
                        align="center"
                        style={{ gap: "0.25rem" }}
                      >
                        <Button.Button
                          variant={currentMode === state.id ? "filled" : "outlined"}
                          size="small"
                          onClick={() => {
                            setCurrentMode(state.id);
                            setActiveRegion(null);
                          }}
                          style={{ minWidth: "80px" }}
                        >
                          {state.name}
                        </Button.Button>
                        {states.length > 2 && (
                          <Button.Button
                            variant="text"
                            size="small"
                            onClick={() => {
                              if (states.length <= 2) return; // Prevent deleting below 2 states
                              setStates((prev) =>
                                prev.filter((s) => s.id !== state.id),
                              );
                              if (currentMode === state.id)
                                setCurrentMode(
                                  states[0].id === state.id
                                    ? states[1].id
                                    : states[0].id,
                                );
                            }}
                            style={{
                              padding: "2px",
                              minWidth: "auto",
                              width: "20px",
                              height: "20px",
                            }}
                          >
                            <Icon.Close style={{ fontSize: "12px" }} />
                          </Button.Button>
                        )}
                      </Flex.Box>
                    ))}
                    <Button.Button
                      variant="outlined"
                      size="small"
                      onClick={() => {
                        const newStateId = `state-${uuid.create()}`;
                        const newState: StateConfig = {
                          id: newStateId,
                          name: `State ${states.length + 1}`,
                          regions: [],
                          color: "#8b5cf6", // Purple default
                        };
                        setStates((prev) => [...prev, newState]);
                        setCurrentMode(newStateId);
                      }}
                      style={{
                        minWidth: "auto",
                        width: "24px",
                        height: "24px",
                        padding: "2px",
                      }}
                    >
                      <Icon.Add style={{ fontSize: "14px" }} />
                    </Button.Button>
                  </Flex.Box>
                  <Button.Button
                    variant={currentMode === "base" ? "filled" : "outlined"}
                    size="small"
                    onClick={() => {
                      setCurrentMode("base");
                      setActiveRegion(null);
                    }}
                    style={{ width: "100%" }}
                  >
                    Base Colors
                  </Button.Button>
                </Flex.Box>
              </Flex.Box>

              {/* Current State Configuration */}
              <Flex.Box y style={{ marginBottom: "1rem" }}>
                <Flex.Box
                  x
                  align="center"
                  justify="between"
                  style={{ marginBottom: "0.5rem" }}
                >
                  {currentMode === "base" ? (
                    <Text.Text level="p">Base Colors</Text.Text>
                  ) : (
                    <Text.Editable
                      level="p"
                      value={
                        states.find((s) => s.id === currentMode)?.name || "Unknown"
                      }
                      onChange={(newName) => {
                        setStates((prev) =>
                          prev.map((state) =>
                            state.id === currentMode
                              ? { ...state, name: newName }
                              : state,
                          ),
                        );
                      }}
                      style={{ fontWeight: "normal" }}
                    />
                  )}
                  <Color.Swatch
                    value={color.hex(
                      currentMode === "base"
                        ? baseColorState.color
                        : states.find((s) => s.id === currentMode)?.color || "#000000",
                    )}
                    onChange={(c) => {
                      const newColor = color.hex(c);
                      if (currentMode === "base")
                        setBaseColorState((prev) => ({ ...prev, color: newColor }));
                      else
                        setStates((prev) =>
                          prev.map((state) =>
                            state.id === currentMode
                              ? { ...state, color: newColor }
                              : state,
                          ),
                        );
                    }}
                    size="small"
                  />
                </Flex.Box>

                {/* Current State Regions */}
                <Flex.Box
                  y
                  style={{
                    border: "1px solid var(--pluto-gray-l5)",
                    borderRadius: "4px",
                    padding: "0.5rem",
                    maxHeight: "300px",
                    overflowY: "auto",
                  }}
                >
                  {(currentMode === "base"
                    ? baseColorState.regions
                    : states.find((s) => s.id === currentMode)?.regions || []
                  ).length === 0 ? (
                    <Text.Text level="small" color={8}>
                      {currentMode === "base"
                        ? "Click SVG elements to override their default colors"
                        : `Click SVG elements to select regions for ${currentMode === "base" ? "base colors" : states.find((s) => s.id === currentMode)?.name || "unknown"} state`}
                    </Text.Text>
                  ) : (
                    (currentMode === "base"
                      ? baseColorState.regions
                      : states.find((s) => s.id === currentMode)?.regions || []
                    ).map((region) => (
                      <Flex.Box
                        key={region.id}
                        y
                        style={{
                          padding: "0.5rem",
                          borderRadius: "4px",
                          backgroundColor:
                            activeRegion === region.id
                              ? "var(--pluto-primary-z)"
                              : "var(--pluto-gray-l3)",
                          marginBottom: "0.5rem",
                          gap: "0.25rem",
                          cursor: "pointer",
                          transition: "background-color 0.2s ease",
                        }}
                        onClick={() => setActiveRegion(region.id)}
                      >
                        {/* Region Header */}
                        <Flex.Box x justify="between" align="center">
                          <Text.Text
                            level="small"
                            style={{
                              fontWeight: "bold",
                              color: activeRegion === region.id ? "white" : "inherit",
                            }}
                          >
                            {region.name || "Element"}
                          </Text.Text>
                          <Button.Button
                            variant="text"
                            size="small"
                            onClick={() => {
                              if (currentMode === "base")
                                setBaseColorState((prev) => ({
                                  ...prev,
                                  regions: prev.regions.filter(
                                    (r) => r.id !== region.id,
                                  ),
                                }));
                              else
                                setStates((prev) =>
                                  prev.map((state) =>
                                    state.id === currentMode
                                      ? {
                                          ...state,
                                          regions: state.regions.filter(
                                            (r) => r.id !== region.id,
                                          ),
                                        }
                                      : state,
                                  ),
                                );

                              // Clear active region if it's being deleted
                              if (activeRegion === region.id) setActiveRegion(null);
                            }}
                            style={{ padding: "2px", minWidth: "auto" }}
                          >
                            <Icon.Close style={{ fontSize: "12px" }} />
                          </Button.Button>
                        </Flex.Box>

                        {/* Custom Colors Toggle */}
                        <Flex.Box x align="center" style={{ gap: "0.5rem" }}>
                          <input
                            type="checkbox"
                            checked={region.useCustomColors || false}
                            onChange={(e) => {
                              const useCustom = e.target.checked;
                              if (currentMode === "base")
                                setBaseColorState((prev) => ({
                                  ...prev,
                                  regions: prev.regions.map((r) =>
                                    r.id === region.id
                                      ? {
                                          ...r,
                                          useCustomColors: useCustom,
                                          fillColor: useCustom
                                            ? r.fillColor || baseColorState.color
                                            : undefined,
                                          strokeColor: useCustom
                                            ? r.strokeColor || "#6366f1"
                                            : undefined,
                                        }
                                      : r,
                                  ),
                                }));
                              else {
                                const currentStateData = states.find(
                                  (s) => s.id === currentMode,
                                );
                                setStates((prev) =>
                                  prev.map((state) =>
                                    state.id === currentMode
                                      ? {
                                          ...state,
                                          regions: state.regions.map((r) =>
                                            r.id === region.id
                                              ? {
                                                  ...r,
                                                  useCustomColors: useCustom,
                                                  fillColor: useCustom
                                                    ? r.fillColor ||
                                                      currentStateData?.color ||
                                                      "#10b981"
                                                    : undefined,
                                                  strokeColor: useCustom
                                                    ? r.strokeColor || "#6366f1"
                                                    : undefined,
                                                }
                                              : r,
                                          ),
                                        }
                                      : state,
                                  ),
                                );
                              }
                            }}
                          />
                          <Text.Text
                            level="small"
                            style={{
                              color: activeRegion === region.id ? "white" : "inherit",
                            }}
                          >
                            Custom Colors
                          </Text.Text>
                        </Flex.Box>

                        {/* Individual Color Controls */}
                        {region.useCustomColors && (
                          <Flex.Box y style={{ gap: "0.25rem" }}>
                            <Flex.Box x align="center" justify="between">
                              <Text.Text
                                level="small"
                                style={{
                                  color:
                                    activeRegion === region.id ? "white" : "inherit",
                                }}
                              >
                                Fill:
                              </Text.Text>
                              <Color.Swatch
                                value={color.hex(
                                  region.fillColor ||
                                    (currentMode === "base"
                                      ? baseColorState.color
                                      : states.find((s) => s.id === currentMode)
                                          ?.color || "#10b981"),
                                )}
                                onChange={(c) => {
                                  const newColor = color.hex(c);
                                  if (currentMode === "base")
                                    setBaseColorState((prev) => ({
                                      ...prev,
                                      regions: prev.regions.map((r) =>
                                        r.id === region.id
                                          ? { ...r, fillColor: newColor }
                                          : r,
                                      ),
                                    }));
                                  else
                                    setStates((prev) =>
                                      prev.map((state) =>
                                        state.id === currentMode
                                          ? {
                                              ...state,
                                              regions: state.regions.map((r) =>
                                                r.id === region.id
                                                  ? { ...r, fillColor: newColor }
                                                  : r,
                                              ),
                                            }
                                          : state,
                                      ),
                                    );
                                }}
                                size="small"
                              />
                            </Flex.Box>
                            <Flex.Box x align="center" justify="between">
                              <Text.Text
                                level="small"
                                style={{
                                  color:
                                    activeRegion === region.id ? "white" : "inherit",
                                }}
                              >
                                Stroke:
                              </Text.Text>
                              <Color.Swatch
                                value={color.hex(region.strokeColor || "#6366f1")}
                                onChange={(c) => {
                                  const newColor = color.hex(c);
                                  if (currentMode === "base")
                                    setBaseColorState((prev) => ({
                                      ...prev,
                                      regions: prev.regions.map((r) =>
                                        r.id === region.id
                                          ? { ...r, strokeColor: newColor }
                                          : r,
                                      ),
                                    }));
                                  else
                                    setStates((prev) =>
                                      prev.map((state) =>
                                        state.id === currentMode
                                          ? {
                                              ...state,
                                              regions: state.regions.map((r) =>
                                                r.id === region.id
                                                  ? { ...r, strokeColor: newColor }
                                                  : r,
                                              ),
                                            }
                                          : state,
                                      ),
                                    );
                                }}
                                size="small"
                              />
                            </Flex.Box>
                          </Flex.Box>
                        )}
                      </Flex.Box>
                    ))
                  )}
                </Flex.Box>
              </Flex.Box>

              {/* State Summary */}
              <Flex.Box y style={{ gap: "0.5rem" }}>
                <Text.Text level="p" style={{ marginBottom: "0.25rem" }}>
                  Summary:
                </Text.Text>
                {states.map((state) => (
                  <Flex.Box key={state.id} x align="center" style={{ gap: "0.5rem" }}>
                    <div
                      style={{
                        width: "12px",
                        height: "12px",
                        backgroundColor: state.color,
                        borderRadius: "2px",
                      }}
                    />
                    <Text.Text level="small">
                      {state.name}: {state.regions.length} regions
                    </Text.Text>
                  </Flex.Box>
                ))}
                <Flex.Box x align="center" style={{ gap: "0.5rem" }}>
                  <div
                    style={{
                      width: "12px",
                      height: "12px",
                      backgroundColor: baseColorState.color,
                      borderRadius: "2px",
                    }}
                  />
                  <Text.Text level="small">
                    Base: {baseColorState.regions.length} regions
                  </Text.Text>
                </Flex.Box>
                {(states.some((s) => s.regions.length > 0) ||
                  baseColorState.regions.length > 0) && (
                  <Text.Text level="small" color={8} style={{ fontStyle: "italic" }}>
                    State colors override base colors
                  </Text.Text>
                )}
              </Flex.Box>
            </Flex.Box>
          </Flex.Box>
        )}
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText />
        <Nav.Bar.End align="center" gap="large">
          <Button.Button
            onClick={() => {
              const symbolConfig = {
                svgContent,
                svgPath,
                states,
                baseColorState,
              };
              console.log("Symbol configuration:", symbolConfig);
              // TODO: Pass this to symbol creation service
            }}
            trigger={Triggers.SAVE}
            disabled={
              states.every((s) => s.regions.length === 0) &&
              baseColorState.regions.length === 0
            }
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
