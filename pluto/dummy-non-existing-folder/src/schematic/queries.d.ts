import { type ontology, type project, schematic } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";
import { Flux } from "../flux";
import { Node } from "./node";
export type RetrieveQuery = schematic.RetrieveSingleParams;
export declare const use: Flux.Use<{
    key: string;
}, schematic.Schematic>, useEnsure: Flux.UseEnsure<{
    key: string;
}>, useTombstone: Flux.UseTombstone<{
    key: string;
}>, createSelector: Flux.CreateSelector<{
    key: string;
}, schematic.Schematic>;
export interface KeyParams {
    key: schematic.Key;
}
export declare const useAllNodes: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    key: string;
    position: {
        x: number;
        y: number;
    };
    zIndex: number;
}[];
export declare const useAllEdges: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    key: string;
    source: {
        node: string;
        param: string;
    };
    target: {
        node: string;
        param: string;
    };
}[];
export declare const useAllConfigs: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => Record<string, {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "cap";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "filter";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flow_straightener";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "heater_element";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "iso_cap";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "iso_filter";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "nozzle";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "orifice";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "orifice_plate";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "strainer";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "strainer_cone";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "thruster";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "vent";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_general";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_electromagnetic";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_variable_area";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_coriolis";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_nozzle";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_venturi";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_ring_piston";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_positive_displacement";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_turbine";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_pulse";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_float_sensor";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flowmeter_orifice";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "box";
    color?: [number, number, number, number] | undefined;
    backgroundColor?: [number, number, number, number] | undefined;
    dimensions: {
        width: number;
        height: number;
    };
    borderRadius: number;
    strokeWidth: number;
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "button";
    size: "huge" | "large" | "medium" | "small" | "tiny";
    level?: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small" | undefined;
    onClickDelay: number;
    commandChannel?: number | undefined;
    mode: "fire" | "momentary" | "pulse";
    color?: [number, number, number, number] | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "circle";
    radius: number;
    color?: [number, number, number, number] | undefined;
    backgroundColor?: [number, number, number, number] | undefined;
    strokeWidth: number;
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    channel?: number | undefined;
    rollingAverage?: number | undefined;
    precision: number;
    notation: "engineering" | "scientific" | "standard";
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "gauge";
    position?: {
        x: number;
        y: number;
    } | undefined;
    color?: [number, number, number, number] | undefined;
    bounds: {
        lower: number;
        upper: number;
    };
    barWidth: number;
    location: {
        x: "center" | "left" | "right";
        y: "bottom" | "center" | "top";
    };
    units: string;
    level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "input";
    size: "huge" | "large" | "medium" | "small" | "tiny";
    commandChannel?: number | undefined;
    dimensions?: {
        width: number;
        height: number;
    } | undefined;
    color?: [number, number, number, number] | undefined;
    disabled: boolean;
    onClickDelay: number;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "light";
    channel?: number | undefined;
    threshold?: {
        lower: number;
        upper: number;
    } | undefined;
    color?: [number, number, number, number] | undefined;
} | {
    variant: "line";
    color?: [number, number, number, number] | undefined;
    start: {
        x: number;
        y: number;
    };
    end: {
        x: number;
        y: number;
    };
    strokeWidth: number;
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    variant: "off_page_reference";
    orientation: "bottom" | "left" | "right" | "top";
    color?: [number, number, number, number] | undefined;
    page?: {
        type: "lineplot" | "log" | "schematic" | "table";
        key: string;
    } | undefined;
    dblClickNavDisabled: boolean;
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "polygon";
    numSides: number;
    sideLength: number;
    rotation: number;
    cornerRounding: number;
    color?: [number, number, number, number] | undefined;
    backgroundColor?: [number, number, number, number] | undefined;
    strokeWidth: number;
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "select";
    size: "huge" | "large" | "medium" | "small" | "tiny";
    commandChannel?: number | undefined;
    color?: [number, number, number, number] | undefined;
    inlineSize: number;
    options: {
        key: string;
        name: string;
        value: number;
        color?: [number, number, number, number] | undefined;
    }[];
    disabled: boolean;
    onClickDelay: number;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    scale: number;
    variant: "scale";
    orientation: "bottom" | "left" | "right" | "top";
    position?: {
        x: number;
        y: number;
    } | undefined;
    dimensions: {
        width: number;
        height: number;
    };
    color?: [number, number, number, number] | undefined;
    indicator: {
        stalenessTimeout: number;
        stalenessColor?: [number, number, number, number] | undefined;
        channel?: number | undefined;
        rollingAverage?: number | undefined;
        precision: number;
        notation: "engineering" | "scientific" | "standard";
        bounds: {
            lower: number;
            upper: number;
        };
        color?: [number, number, number, number] | undefined;
        axisColor?: [number, number, number, number] | undefined;
        textColor?: [number, number, number, number] | undefined;
        units: string;
        fillHidden: boolean;
        caretHidden: boolean;
        scaleHidden: boolean;
        side: "bottom" | "left" | "right" | "top";
        caretSide: "bottom" | "left" | "right" | "top";
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
    };
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "setpoint";
    size: "huge" | "large" | "medium" | "small" | "tiny";
    commandChannel?: number | undefined;
    dimensions?: {
        width: number;
        height: number;
    } | undefined;
    color?: [number, number, number, number] | undefined;
    units: string;
    disabled: boolean;
    onClickDelay: number;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "state_indicator";
    channel?: number | undefined;
    color?: [number, number, number, number] | undefined;
    inlineSize: number;
    options: {
        key: string;
        name: string;
        value: number;
        color?: [number, number, number, number] | undefined;
    }[];
    size: "huge" | "large" | "medium" | "small" | "tiny";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "string_display";
    color?: [number, number, number, number] | undefined;
    textColor?: [number, number, number, number] | undefined;
    tooltip: string[];
    inlineSize: number;
    channel?: number | undefined;
    level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "switch";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "text_box";
    color?: [number, number, number, number] | undefined;
    width: number;
    align: "center" | "end" | "start" | "stretch";
    autoFitDisabled: boolean;
    level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
    value: string;
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    channel?: number | undefined;
    rollingAverage?: number | undefined;
    precision: number;
    notation: "engineering" | "scientific" | "standard";
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "value";
    position?: {
        x: number;
        y: number;
    } | undefined;
    color?: [number, number, number, number] | undefined;
    textColor?: [number, number, number, number] | undefined;
    tooltip: string[];
    redline: {
        bounds: {
            lower: number;
            upper: number;
        };
        gradient: {
            key: string;
            color: string | {
                rgba255: [number, number, number, number];
            } | {
                r: number;
                g: number;
                b: number;
                a: number;
            } | {
                rgba255: [number, number, number, number];
            } | {
                r: number;
                g: number;
                b: number;
                a: number;
            } | [number, number, number] | [number, number, number, number];
            position: number;
            switched?: boolean | undefined;
        }[];
    };
    units: string;
    inlineSize: number;
    level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
    location: {
        x: "center" | "left" | "right";
        y: "bottom" | "center" | "top";
    };
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "agitator";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "cross_beam_agitator";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "flat_blade_agitator";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "heat_exchanger_general";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "heat_exchanger_m";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "heat_exchanger_straight_tube";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "helical_agitator";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "paddle_agitator";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "propeller_agitator";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "rotary_mixer";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "static_mixer";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "cavity_pump";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "centrifugal_compressor";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "compressor";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "diaphragm_pump";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "ejection_pump";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "ejector_compressor";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "liquid_ring_compressor";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "piston_pump";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "pump";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "roller_vane_compressor";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "screw_pump";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "turbo_compressor";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "vacuum_pump";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "burst_disc";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flame_arrestor";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flame_arrestor_detonation";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flame_arrestor_explosion";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flame_arrestor_fire_res";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "flame_arrestor_fire_res_detonation";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "iso_burst_disc";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "angled_valve";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    enabled: boolean;
    clickable: boolean;
    color?: [number, number, number, number] | undefined;
    variant: "angled_relief_valve";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    enabled: boolean;
    clickable: boolean;
    color?: [number, number, number, number] | undefined;
    variant: "angled_spring_loaded_relief_valve";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "ball_valve";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    enabled: boolean;
    clickable: boolean;
    color?: [number, number, number, number] | undefined;
    variant: "breather_valve";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "butterfly_valve_one";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "butterfly_valve_two";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "check_valve";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "check_valve_with_arrow";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "electric_regulator";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "electric_regulator_motorized";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "four_way_valve";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "gate_valve";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "iso_check_valve";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    enabled: boolean;
    clickable: boolean;
    color?: [number, number, number, number] | undefined;
    variant: "manual_valve";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    enabled: boolean;
    clickable: boolean;
    color?: [number, number, number, number] | undefined;
    variant: "needle_valve";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "regulator";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "regulator_manual";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    enabled: boolean;
    clickable: boolean;
    color?: [number, number, number, number] | undefined;
    variant: "relief_valve";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "solenoid_valve";
    normallyOpen: boolean;
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    enabled: boolean;
    clickable: boolean;
    color?: [number, number, number, number] | undefined;
    variant: "spring_loaded_relief_valve";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "three_way_valve";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "three_way_ball_valve";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    color?: [number, number, number, number] | undefined;
    variant: "valve";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "cross_junction";
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "cylinder";
    dimensions: {
        width: number;
        height: number;
    };
    borderRadius?: {
        topLeft: {
            x: number;
            y: number;
        };
        topRight: {
            x: number;
            y: number;
        };
        bottomLeft: {
            x: number;
            y: number;
        };
        bottomRight: {
            x: number;
            y: number;
        };
    } | undefined;
    color?: [number, number, number, number] | undefined;
    backgroundColor?: [number, number, number, number] | undefined;
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "tank";
    position?: {
        x: number;
        y: number;
    } | undefined;
    color?: [number, number, number, number] | undefined;
    backgroundColor?: [number, number, number, number] | undefined;
    dimensions: {
        width: number;
        height: number;
    };
    borderRadius: {
        topLeft: {
            x: number;
            y: number;
        };
        topRight: {
            x: number;
            y: number;
        };
        bottomLeft: {
            x: number;
            y: number;
        };
        bottomRight: {
            x: number;
            y: number;
        };
    };
    fill: {
        stalenessTimeout: number;
        stalenessColor?: [number, number, number, number] | undefined;
        channel?: number | undefined;
        rollingAverage?: number | undefined;
        precision: number;
        notation: "engineering" | "scientific" | "standard";
        bounds: {
            lower: number;
            upper: number;
        };
        color?: [number, number, number, number] | undefined;
        axisColor?: [number, number, number, number] | undefined;
        textColor?: [number, number, number, number] | undefined;
        units: string;
        fillHidden: boolean;
        caretHidden: boolean;
        scaleHidden: boolean;
        side: "bottom" | "left" | "right" | "top";
        caretSide: "bottom" | "left" | "right" | "top";
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
    };
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    color?: [number, number, number, number] | undefined;
    variant: "t_junction";
} | {
    stalenessTimeout: number;
    stalenessColor?: [number, number, number, number] | undefined;
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    stateChannel?: number | undefined;
    commandChannel?: number | undefined;
    control?: {
        authority?: number | undefined;
        hidden: boolean;
        chipHidden: boolean;
        indicatorHidden: boolean;
        orientation: "bottom" | "center" | "left" | "right" | "top";
    } | undefined;
    onClickDelay: number;
    variant: "custom_actuator";
    specKey: string;
    color?: [number, number, number, number] | undefined;
    stateOverrides: {
        key: string;
        name: string;
        regions: {
            key: string;
            name: string;
            selectors: string[];
            strokeColor?: [number, number, number, number] | undefined;
            fillColor?: [number, number, number, number] | undefined;
        }[];
    }[];
} | {
    label: {
        label: string;
        level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        orientation: "bottom" | "center" | "left" | "right" | "top";
        direction: "x" | "y";
        maxInlineSize: number;
        align: "center" | "end" | "start" | "stretch";
    };
    orientation: "bottom" | "left" | "right" | "top";
    scale: number;
    variant: "custom_static";
    specKey: string;
    color?: [number, number, number, number] | undefined;
    stateOverrides: {
        key: string;
        name: string;
        regions: {
            key: string;
            name: string;
            selectors: string[];
            strokeColor?: [number, number, number, number] | undefined;
            fillColor?: [number, number, number, number] | undefined;
        }[];
    }[];
} | {
    variant: "group_box";
    members: string[];
    locked: boolean;
} | {
    color?: [number, number, number, number] | undefined;
    segments: {
        direction: "x" | "y";
        length: number;
    }[];
    variant: "pipe";
} | {
    color?: [number, number, number, number] | undefined;
    segments: {
        direction: "x" | "y";
        length: number;
    }[];
    variant: "electric";
} | {
    color?: [number, number, number, number] | undefined;
    segments: {
        direction: "x" | "y";
        length: number;
    }[];
    variant: "secondary";
} | {
    color?: [number, number, number, number] | undefined;
    segments: {
        direction: "x" | "y";
        length: number;
    }[];
    variant: "jacketed";
} | {
    color?: [number, number, number, number] | undefined;
    segments: {
        direction: "x" | "y";
        length: number;
    }[];
    variant: "hydraulic";
} | {
    color?: [number, number, number, number] | undefined;
    segments: {
        direction: "x" | "y";
        length: number;
    }[];
    variant: "pneumatic";
} | {
    color?: [number, number, number, number] | undefined;
    segments: {
        direction: "x" | "y";
        length: number;
    }[];
    variant: "data";
}>;
export declare const useParentOf: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => Map<string, string>;
export interface ConfigParams extends KeyParams {
    elKey: string;
}
export declare const useElementConfig: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<ConfigParams, "key">) => schematic.ElementConfig | undefined;
export interface ConfigsParams extends KeyParams {
    keys: string[];
}
export declare const useConfigs: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<ConfigsParams, "key">) => Map<string, schematic.ElementConfig>;
export interface NodesParams extends KeyParams {
    keys: string[];
}
export declare const useNodes: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<NodesParams, "key">) => schematic.Node[];
export declare const useIsSnapshot: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => boolean;
export declare const useName: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => string;
export type DeleteParams = schematic.Key | schematic.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, import("zod").ZodNever>;
export interface CopyParams extends schematic.CopyParams {
}
export declare const useCopy: Flux.UseUpdate<CopyParams, schematic.Schematic, import("zod").ZodNever>;
export interface UseCreateParams extends schematic.New {
    project?: project.Key;
}
export declare const useCreate: Flux.UseUpdate<UseCreateParams, schematic.Schematic, import("zod").ZodNever>;
export interface SnapshotPair extends Pick<schematic.Schematic, "key" | "name"> {
}
export interface SnapshotParams {
    schematics: SnapshotPair | SnapshotPair[];
    parentID: ontology.ID;
}
export declare const useSnapshot: Flux.UseUpdate<SnapshotParams, SnapshotParams, import("zod").ZodNever>;
export declare const useDispatch: () => Flux.UseDispatchReturn<string, schematic.Action>, useUndoBase: (params: {
    key: string;
}) => Flux.UseUndoReturn, useRedoBase: (params: import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>) => Flux.UseRedoReturn, useSingleDispatchBase: (params: import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>) => (action: schematic.Action[] | schematic.Action) => void;
export declare const useSingleDispatch: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>, "key"> | undefined) => (action: schematic.Action[] | schematic.Action) => void;
export declare const useUndo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => Flux.UseUndoReturn;
export declare const useRedo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>, "key"> | undefined) => Flux.UseRedoReturn;
export interface RenameParams extends Pick<schematic.Schematic, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, import("zod").ZodNever>;
export interface AddNodeProps<V extends Node.Variant = Node.Variant> {
    key: string;
    position?: xy.XY;
    config: Node.Input<V>;
}
export declare const useAddNode: () => <V extends Node.Variant>({ key, position, config: input }: AddNodeProps<V>) => void;
/**
 * useGroup returns a callback that groups the given selection, dispatched as a
 * single undoable step. Returns the keys to select, the new group first, or null
 * when the selection cannot be grouped.
 */
export declare const useGroup: () => ((selected: readonly string[]) => string[] | null);
/**
 * useUngroup returns a callback that dissolves the groups the given selection
 * resolves to, dispatched as a single undoable step. Returns the freed member
 * keys, or null when the selection touches no group.
 */
export declare const useUngroup: () => ((selected: readonly string[]) => string[] | null);
//# sourceMappingURL=queries.d.ts.map