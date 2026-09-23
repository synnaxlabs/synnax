import { schematic } from "@synnaxlabs/client";
import { type z } from "zod";
import { CUSTOM_ACTUATOR_VARIANT, CUSTOM_STATIC_VARIANT } from "./custom/configs";
import { type Spec } from "./spec";
export declare const REGISTRY: {
    readonly cap: Spec<"cap", import("./common/create").StaticConfig<"cap">>;
    readonly filter: Spec<"filter", import("./common/create").StaticConfig<"filter">>;
    readonly flow_straightener: Spec<"flow_straightener", import("./common/create").StaticConfig<"flow_straightener">>;
    readonly heater_element: Spec<"heater_element", import("./common/create").StaticConfig<"heater_element">>;
    readonly iso_cap: Spec<"iso_cap", import("./common/create").StaticConfig<"iso_cap">>;
    readonly iso_filter: Spec<"iso_filter", import("./common/create").StaticConfig<"iso_filter">>;
    readonly nozzle: Spec<"nozzle", import("./common/create").StaticConfig<"nozzle">>;
    readonly orifice: Spec<"orifice", import("./common/create").StaticConfig<"orifice">>;
    readonly orifice_plate: Spec<"orifice_plate", import("./common/create").StaticConfig<"orifice_plate">>;
    readonly strainer: Spec<"strainer", import("./common/create").StaticConfig<"strainer">>;
    readonly strainer_cone: Spec<"strainer_cone", import("./common/create").StaticConfig<"strainer_cone">>;
    readonly thruster: Spec<"thruster", import("./common/create").ToggleSymbolConfig<"thruster">>;
    readonly vent: Spec<"vent", import("./common/create").StaticConfig<"vent">>;
    readonly flowmeter_general: Spec<"flowmeter_general", import("./common/create").StaticConfig<"flowmeter_general">>;
    readonly flowmeter_electromagnetic: Spec<"flowmeter_electromagnetic", import("./common/create").StaticConfig<"flowmeter_electromagnetic">>;
    readonly flowmeter_variable_area: Spec<"flowmeter_variable_area", import("./common/create").StaticConfig<"flowmeter_variable_area">>;
    readonly flowmeter_coriolis: Spec<"flowmeter_coriolis", import("./common/create").StaticConfig<"flowmeter_coriolis">>;
    readonly flowmeter_nozzle: Spec<"flowmeter_nozzle", import("./common/create").StaticConfig<"flowmeter_nozzle">>;
    readonly flowmeter_venturi: Spec<"flowmeter_venturi", import("./common/create").StaticConfig<"flowmeter_venturi">>;
    readonly flowmeter_ring_piston: Spec<"flowmeter_ring_piston", import("./common/create").StaticConfig<"flowmeter_ring_piston">>;
    readonly flowmeter_positive_displacement: Spec<"flowmeter_positive_displacement", import("./common/create").StaticConfig<"flowmeter_positive_displacement">>;
    readonly flowmeter_turbine: Spec<"flowmeter_turbine", import("./common/create").StaticConfig<"flowmeter_turbine">>;
    readonly flowmeter_pulse: Spec<"flowmeter_pulse", import("./common/create").StaticConfig<"flowmeter_pulse">>;
    readonly flowmeter_float_sensor: Spec<"flowmeter_float_sensor", import("./common/create").StaticConfig<"flowmeter_float_sensor">>;
    readonly flowmeter_orifice: Spec<"flowmeter_orifice", import("./common/create").StaticConfig<"flowmeter_orifice">>;
    readonly box: Spec<"box", schematic.BoxNodeConfig>;
    readonly button: Spec<"button", schematic.ButtonNodeConfig>;
    readonly circle: Spec<"circle", schematic.CircleNodeConfig>;
    readonly gauge: Spec<"gauge", schematic.GaugeNodeConfig>;
    readonly input: Spec<"input", schematic.InputNodeConfig>;
    readonly light: Spec<"light", schematic.LightNodeConfig>;
    readonly line: Spec<"line", schematic.LineNodeConfig>;
    readonly off_page_reference: Spec<"off_page_reference", schematic.OffPageReferenceNodeConfig>;
    readonly polygon: Spec<"polygon", schematic.PolygonNodeConfig>;
    readonly scale: Spec<"scale", schematic.ScaleNodeConfig>;
    readonly select: Spec<"select", schematic.SelectNodeConfig>;
    readonly setpoint: Spec<"setpoint", schematic.SetpointNodeConfig>;
    readonly state_indicator: Spec<"state_indicator", schematic.StateIndicatorNodeConfig>;
    readonly string_display: Spec<"string_display", schematic.StringDisplayNodeConfig>;
    readonly switch: Spec<"switch", schematic.SwitchNodeConfig>;
    readonly text_box: Spec<"text_box", schematic.TextBoxNodeConfig>;
    readonly value: Spec<"value", schematic.ValueNodeConfig>;
    readonly agitator: Spec<"agitator", import("./common/create").ToggleSymbolConfig<"agitator">>;
    readonly cross_beam_agitator: Spec<"cross_beam_agitator", import("./common/create").ToggleSymbolConfig<"cross_beam_agitator">>;
    readonly flat_blade_agitator: Spec<"flat_blade_agitator", import("./common/create").ToggleSymbolConfig<"flat_blade_agitator">>;
    readonly heat_exchanger_general: Spec<"heat_exchanger_general", import("./common/create").StaticConfig<"heat_exchanger_general">>;
    readonly heat_exchanger_m: Spec<"heat_exchanger_m", import("./common/create").StaticConfig<"heat_exchanger_m">>;
    readonly heat_exchanger_straight_tube: Spec<"heat_exchanger_straight_tube", import("./common/create").StaticConfig<"heat_exchanger_straight_tube">>;
    readonly helical_agitator: Spec<"helical_agitator", import("./common/create").ToggleSymbolConfig<"helical_agitator">>;
    readonly paddle_agitator: Spec<"paddle_agitator", import("./common/create").ToggleSymbolConfig<"paddle_agitator">>;
    readonly propeller_agitator: Spec<"propeller_agitator", import("./common/create").ToggleSymbolConfig<"propeller_agitator">>;
    readonly rotary_mixer: Spec<"rotary_mixer", import("./common/create").ToggleSymbolConfig<"rotary_mixer">>;
    readonly static_mixer: Spec<"static_mixer", import("./common/create").StaticConfig<"static_mixer">>;
    readonly cavity_pump: Spec<"cavity_pump", import("./common/create").ToggleSymbolConfig<"cavity_pump">>;
    readonly centrifugal_compressor: Spec<"centrifugal_compressor", import("./common/create").ToggleSymbolConfig<"centrifugal_compressor">>;
    readonly compressor: Spec<"compressor", import("./common/create").ToggleSymbolConfig<"compressor">>;
    readonly diaphragm_pump: Spec<"diaphragm_pump", import("./common/create").ToggleSymbolConfig<"diaphragm_pump">>;
    readonly ejection_pump: Spec<"ejection_pump", import("./common/create").ToggleSymbolConfig<"ejection_pump">>;
    readonly ejector_compressor: Spec<"ejector_compressor", import("./common/create").ToggleSymbolConfig<"ejector_compressor">>;
    readonly liquid_ring_compressor: Spec<"liquid_ring_compressor", import("./common/create").ToggleSymbolConfig<"liquid_ring_compressor">>;
    readonly piston_pump: Spec<"piston_pump", import("./common/create").ToggleSymbolConfig<"piston_pump">>;
    readonly pump: Spec<"pump", import("./common/create").ToggleSymbolConfig<"pump">>;
    readonly roller_vane_compressor: Spec<"roller_vane_compressor", import("./common/create").ToggleSymbolConfig<"roller_vane_compressor">>;
    readonly screw_pump: Spec<"screw_pump", import("./common/create").ToggleSymbolConfig<"screw_pump">>;
    readonly turbo_compressor: Spec<"turbo_compressor", import("./common/create").ToggleSymbolConfig<"turbo_compressor">>;
    readonly vacuum_pump: Spec<"vacuum_pump", import("./common/create").ToggleSymbolConfig<"vacuum_pump">>;
    readonly burst_disc: Spec<"burst_disc", import("./common/create").StaticConfig<"burst_disc">>;
    readonly flame_arrestor: Spec<"flame_arrestor", import("./common/create").StaticConfig<"flame_arrestor">>;
    readonly flame_arrestor_detonation: Spec<"flame_arrestor_detonation", import("./common/create").StaticConfig<"flame_arrestor_detonation">>;
    readonly flame_arrestor_explosion: Spec<"flame_arrestor_explosion", import("./common/create").StaticConfig<"flame_arrestor_explosion">>;
    readonly flame_arrestor_fire_res: Spec<"flame_arrestor_fire_res", import("./common/create").StaticConfig<"flame_arrestor_fire_res">>;
    readonly flame_arrestor_fire_res_detonation: Spec<"flame_arrestor_fire_res_detonation", import("./common/create").StaticConfig<"flame_arrestor_fire_res_detonation">>;
    readonly iso_burst_disc: Spec<"iso_burst_disc", import("./common/create").StaticConfig<"iso_burst_disc">>;
    readonly angled_valve: Spec<"angled_valve", import("./common/create").ToggleSymbolConfig<"angled_valve">>;
    readonly angled_relief_valve: Spec<"angled_relief_valve", import("./common/create").DummyToggleConfig<"angled_relief_valve">>;
    readonly angled_spring_loaded_relief_valve: Spec<"angled_spring_loaded_relief_valve", import("./common/create").DummyToggleConfig<"angled_spring_loaded_relief_valve">>;
    readonly ball_valve: Spec<"ball_valve", import("./common/create").ToggleSymbolConfig<"ball_valve">>;
    readonly breather_valve: Spec<"breather_valve", import("./common/create").DummyToggleConfig<"breather_valve">>;
    readonly butterfly_valve_one: Spec<"butterfly_valve_one", import("./common/create").ToggleSymbolConfig<"butterfly_valve_one">>;
    readonly butterfly_valve_two: Spec<"butterfly_valve_two", import("./common/create").ToggleSymbolConfig<"butterfly_valve_two">>;
    readonly check_valve: Spec<"check_valve", import("./common/create").StaticConfig<"check_valve">>;
    readonly check_valve_with_arrow: Spec<"check_valve_with_arrow", import("./common/create").StaticConfig<"check_valve_with_arrow">>;
    readonly electric_regulator: Spec<"electric_regulator", import("./common/create").StaticConfig<"electric_regulator">>;
    readonly electric_regulator_motorized: Spec<"electric_regulator_motorized", import("./common/create").StaticConfig<"electric_regulator_motorized">>;
    readonly four_way_valve: Spec<"four_way_valve", import("./common/create").ToggleSymbolConfig<"four_way_valve">>;
    readonly gate_valve: Spec<"gate_valve", import("./common/create").ToggleSymbolConfig<"gate_valve">>;
    readonly iso_check_valve: Spec<"iso_check_valve", import("./common/create").StaticConfig<"iso_check_valve">>;
    readonly manual_valve: Spec<"manual_valve", import("./common/create").DummyToggleConfig<"manual_valve">>;
    readonly needle_valve: Spec<"needle_valve", import("./common/create").DummyToggleConfig<"needle_valve">>;
    readonly regulator: Spec<"regulator", import("./common/create").StaticConfig<"regulator">>;
    readonly regulator_manual: Spec<"regulator_manual", import("./common/create").StaticConfig<"regulator_manual">>;
    readonly relief_valve: Spec<"relief_valve", import("./common/create").DummyToggleConfig<"relief_valve">>;
    readonly solenoid_valve: Spec<"solenoid_valve", schematic.SolenoidValveNodeConfig>;
    readonly spring_loaded_relief_valve: Spec<"spring_loaded_relief_valve", import("./common/create").DummyToggleConfig<"spring_loaded_relief_valve">>;
    readonly three_way_valve: Spec<"three_way_valve", import("./common/create").ToggleSymbolConfig<"three_way_valve">>;
    readonly three_way_ball_valve: Spec<"three_way_ball_valve", import("./common/create").ToggleSymbolConfig<"three_way_ball_valve">>;
    readonly valve: Spec<"valve", import("./common/create").ToggleSymbolConfig<"valve">>;
    readonly cross_junction: Spec<"cross_junction", schematic.CrossJunctionNodeConfig>;
    readonly cylinder: Spec<"cylinder", schematic.CylinderNodeConfig>;
    readonly tank: Spec<"tank", schematic.TankNodeConfig>;
    readonly t_junction: Spec<"t_junction", schematic.TJunctionNodeConfig>;
    readonly custom_actuator: Spec<"custom_actuator", import("./custom/configs").CustomActuatorConfig>;
    readonly custom_static: Spec<"custom_static", import("./custom/configs").CustomStaticConfig>;
    readonly group_box: Spec<"group_box", schematic.GroupBoxNodeConfig>;
};
export declare const variantZ: z.ZodEnum<{
    agitator: "agitator";
    angled_relief_valve: "angled_relief_valve";
    angled_spring_loaded_relief_valve: "angled_spring_loaded_relief_valve";
    angled_valve: "angled_valve";
    ball_valve: "ball_valve";
    box: "box";
    breather_valve: "breather_valve";
    burst_disc: "burst_disc";
    butterfly_valve_one: "butterfly_valve_one";
    butterfly_valve_two: "butterfly_valve_two";
    button: "button";
    cap: "cap";
    cavity_pump: "cavity_pump";
    centrifugal_compressor: "centrifugal_compressor";
    check_valve: "check_valve";
    check_valve_with_arrow: "check_valve_with_arrow";
    circle: "circle";
    compressor: "compressor";
    cross_beam_agitator: "cross_beam_agitator";
    cross_junction: "cross_junction";
    custom_actuator: "custom_actuator";
    custom_static: "custom_static";
    cylinder: "cylinder";
    diaphragm_pump: "diaphragm_pump";
    ejection_pump: "ejection_pump";
    ejector_compressor: "ejector_compressor";
    electric_regulator: "electric_regulator";
    electric_regulator_motorized: "electric_regulator_motorized";
    filter: "filter";
    flame_arrestor: "flame_arrestor";
    flame_arrestor_detonation: "flame_arrestor_detonation";
    flame_arrestor_explosion: "flame_arrestor_explosion";
    flame_arrestor_fire_res: "flame_arrestor_fire_res";
    flame_arrestor_fire_res_detonation: "flame_arrestor_fire_res_detonation";
    flat_blade_agitator: "flat_blade_agitator";
    flow_straightener: "flow_straightener";
    flowmeter_coriolis: "flowmeter_coriolis";
    flowmeter_electromagnetic: "flowmeter_electromagnetic";
    flowmeter_float_sensor: "flowmeter_float_sensor";
    flowmeter_general: "flowmeter_general";
    flowmeter_nozzle: "flowmeter_nozzle";
    flowmeter_orifice: "flowmeter_orifice";
    flowmeter_positive_displacement: "flowmeter_positive_displacement";
    flowmeter_pulse: "flowmeter_pulse";
    flowmeter_ring_piston: "flowmeter_ring_piston";
    flowmeter_turbine: "flowmeter_turbine";
    flowmeter_variable_area: "flowmeter_variable_area";
    flowmeter_venturi: "flowmeter_venturi";
    four_way_valve: "four_way_valve";
    gate_valve: "gate_valve";
    gauge: "gauge";
    group_box: "group_box";
    heat_exchanger_general: "heat_exchanger_general";
    heat_exchanger_m: "heat_exchanger_m";
    heat_exchanger_straight_tube: "heat_exchanger_straight_tube";
    heater_element: "heater_element";
    helical_agitator: "helical_agitator";
    input: "input";
    iso_burst_disc: "iso_burst_disc";
    iso_cap: "iso_cap";
    iso_check_valve: "iso_check_valve";
    iso_filter: "iso_filter";
    light: "light";
    line: "line";
    liquid_ring_compressor: "liquid_ring_compressor";
    manual_valve: "manual_valve";
    needle_valve: "needle_valve";
    nozzle: "nozzle";
    off_page_reference: "off_page_reference";
    orifice: "orifice";
    orifice_plate: "orifice_plate";
    paddle_agitator: "paddle_agitator";
    piston_pump: "piston_pump";
    polygon: "polygon";
    propeller_agitator: "propeller_agitator";
    pump: "pump";
    regulator: "regulator";
    regulator_manual: "regulator_manual";
    relief_valve: "relief_valve";
    roller_vane_compressor: "roller_vane_compressor";
    rotary_mixer: "rotary_mixer";
    scale: "scale";
    screw_pump: "screw_pump";
    select: "select";
    setpoint: "setpoint";
    solenoid_valve: "solenoid_valve";
    spring_loaded_relief_valve: "spring_loaded_relief_valve";
    state_indicator: "state_indicator";
    static_mixer: "static_mixer";
    strainer: "strainer";
    strainer_cone: "strainer_cone";
    string_display: "string_display";
    switch: "switch";
    t_junction: "t_junction";
    tank: "tank";
    text_box: "text_box";
    three_way_ball_valve: "three_way_ball_valve";
    three_way_valve: "three_way_valve";
    thruster: "thruster";
    turbo_compressor: "turbo_compressor";
    vacuum_pump: "vacuum_pump";
    value: "value";
    valve: "valve";
    vent: "vent";
}>;
export type Variant = schematic.NodeConfigType;
export declare const configZ: z.ZodDiscriminatedUnion<[z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"cap">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"filter">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flow_straightener">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"heater_element">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"iso_cap">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"iso_filter">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"nozzle">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"orifice">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"orifice_plate">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"strainer">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"strainer_cone">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"thruster">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"vent">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_general">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_electromagnetic">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_variable_area">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_coriolis">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_nozzle">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_venturi">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_ring_piston">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_positive_displacement">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_turbine">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_pulse">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_float_sensor">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flowmeter_orifice">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"box">;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    backgroundColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    dimensions: z.ZodPrefault<z.ZodObject<{
        width: z.ZodNumber;
        height: z.ZodNumber;
    }, z.core.$strip>>;
    borderRadius: z.ZodDefault<z.ZodNumber>;
    strokeWidth: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"button">;
    size: z.ZodDefault<z.ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
    level: z.ZodOptional<z.ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    mode: z.ZodDefault<z.ZodEnum<{
        fire: "fire";
        momentary: "momentary";
        pulse: "pulse";
    }>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"circle">;
    radius: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    backgroundColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    strokeWidth: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    channel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    rollingAverage: z.ZodOptional<z.ZodInt32>;
    precision: z.ZodDefault<z.ZodNumber>;
    notation: z.ZodDefault<z.ZodEnum<{
        engineering: "engineering";
        scientific: "scientific";
        standard: "standard";
    }>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"gauge">;
    position: z.ZodOptional<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    bounds: z.ZodPrefault<z.ZodObject<{
        lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
    }, z.core.$strip>>;
    barWidth: z.ZodDefault<z.ZodNumber>;
    location: z.ZodPrefault<z.ZodObject<{
        x: z.ZodEnum<{
            center: "center";
            left: "left";
            right: "right";
        }>;
        y: z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            top: "top";
        }>;
    }, z.core.$strip>>;
    units: z.ZodDefault<z.ZodString>;
    level: z.ZodDefault<z.ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"input">;
    size: z.ZodDefault<z.ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    dimensions: z.ZodOptional<z.ZodObject<{
        width: z.ZodNumber;
        height: z.ZodNumber;
    }, z.core.$strip>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    disabled: z.ZodDefault<z.ZodBoolean>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"light">;
    channel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    threshold: z.ZodOptional<z.ZodObject<{
        lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
    }, z.core.$strip>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
}, z.core.$strip>, z.ZodObject<{
    variant: z.ZodLiteral<"line">;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    start: z.ZodPrefault<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
    end: z.ZodPrefault<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
    strokeWidth: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    variant: z.ZodLiteral<"off_page_reference">;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    page: z.ZodOptional<z.ZodObject<{
        type: z.ZodEnum<{
            lineplot: "lineplot";
            log: "log";
            schematic: "schematic";
            table: "table";
        }>;
        key: z.ZodString;
    }, z.core.$strip>>;
    dblClickNavDisabled: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"polygon">;
    numSides: z.ZodDefault<z.ZodNumber>;
    sideLength: z.ZodDefault<z.ZodNumber>;
    rotation: z.ZodDefault<z.ZodNumber>;
    cornerRounding: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    backgroundColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    strokeWidth: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"select">;
    size: z.ZodDefault<z.ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    inlineSize: z.ZodDefault<z.ZodNumber>;
    options: z.ZodDefault<z.ZodArray<z.ZodObject<{
        key: z.ZodString;
        name: z.ZodString;
        value: z.ZodNumber;
        color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
    }, z.core.$strip>>>;
    disabled: z.ZodDefault<z.ZodBoolean>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"scale">;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    position: z.ZodOptional<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
    dimensions: z.ZodPrefault<z.ZodObject<{
        width: z.ZodNumber;
        height: z.ZodNumber;
    }, z.core.$strip>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    indicator: z.ZodPrefault<z.ZodObject<{
        stalenessTimeout: z.ZodDefault<z.ZodNumber>;
        stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
        channel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
        rollingAverage: z.ZodOptional<z.ZodInt32>;
        precision: z.ZodDefault<z.ZodNumber>;
        notation: z.ZodDefault<z.ZodEnum<{
            engineering: "engineering";
            scientific: "scientific";
            standard: "standard";
        }>>;
        bounds: z.ZodPrefault<z.ZodObject<{
            lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        }, z.core.$strip>>;
        color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
        axisColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
        textColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
        units: z.ZodDefault<z.ZodString>;
        fillHidden: z.ZodDefault<z.ZodBoolean>;
        caretHidden: z.ZodDefault<z.ZodBoolean>;
        scaleHidden: z.ZodDefault<z.ZodBoolean>;
        side: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        caretSide: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
    }, z.core.$strip>>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"setpoint">;
    size: z.ZodDefault<z.ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    dimensions: z.ZodOptional<z.ZodObject<{
        width: z.ZodNumber;
        height: z.ZodNumber;
    }, z.core.$strip>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    units: z.ZodDefault<z.ZodString>;
    disabled: z.ZodDefault<z.ZodBoolean>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"state_indicator">;
    channel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    inlineSize: z.ZodDefault<z.ZodNumber>;
    options: z.ZodDefault<z.ZodArray<z.ZodObject<{
        key: z.ZodString;
        name: z.ZodString;
        value: z.ZodNumber;
        color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
    }, z.core.$strip>>>;
    size: z.ZodDefault<z.ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"string_display">;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    textColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    tooltip: z.ZodDefault<z.ZodArray<z.ZodString>>;
    inlineSize: z.ZodDefault<z.ZodNumber>;
    channel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    level: z.ZodDefault<z.ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"switch">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"text_box">;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    width: z.ZodDefault<z.ZodNumber>;
    align: z.ZodDefault<z.ZodEnum<{
        center: "center";
        end: "end";
        start: "start";
        stretch: "stretch";
    }>>;
    autoFitDisabled: z.ZodDefault<z.ZodBoolean>;
    level: z.ZodDefault<z.ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    value: z.ZodDefault<z.ZodString>;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    channel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    rollingAverage: z.ZodOptional<z.ZodInt32>;
    precision: z.ZodDefault<z.ZodNumber>;
    notation: z.ZodDefault<z.ZodEnum<{
        engineering: "engineering";
        scientific: "scientific";
        standard: "standard";
    }>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"value">;
    position: z.ZodOptional<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    textColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    tooltip: z.ZodDefault<z.ZodArray<z.ZodString>>;
    redline: z.ZodPrefault<z.ZodObject<{
        bounds: z.ZodObject<{
            lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        }, z.core.$strip>;
        gradient: z.ZodDefault<z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            color: z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodNumber;
            }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
            }, z.core.$strip>]>;
            position: z.ZodNumber;
            switched: z.ZodOptional<z.ZodBoolean>;
        }, z.core.$strip>>>;
    }, z.core.$strip>>;
    units: z.ZodDefault<z.ZodString>;
    inlineSize: z.ZodDefault<z.ZodNumber>;
    level: z.ZodDefault<z.ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    location: z.ZodPrefault<z.ZodObject<{
        x: z.ZodEnum<{
            center: "center";
            left: "left";
            right: "right";
        }>;
        y: z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            top: "top";
        }>;
    }, z.core.$strip>>;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"agitator">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"cross_beam_agitator">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flat_blade_agitator">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"heat_exchanger_general">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"heat_exchanger_m">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"heat_exchanger_straight_tube">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"helical_agitator">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"paddle_agitator">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"propeller_agitator">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"rotary_mixer">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"static_mixer">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"cavity_pump">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"centrifugal_compressor">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"compressor">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"diaphragm_pump">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"ejection_pump">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"ejector_compressor">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"liquid_ring_compressor">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"piston_pump">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"pump">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"roller_vane_compressor">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"screw_pump">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"turbo_compressor">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"vacuum_pump">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"burst_disc">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flame_arrestor">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flame_arrestor_detonation">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flame_arrestor_explosion">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flame_arrestor_fire_res">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"flame_arrestor_fire_res_detonation">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"iso_burst_disc">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"angled_valve">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    enabled: z.ZodDefault<z.ZodBoolean>;
    clickable: z.ZodDefault<z.ZodBoolean>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"angled_relief_valve">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    enabled: z.ZodDefault<z.ZodBoolean>;
    clickable: z.ZodDefault<z.ZodBoolean>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"angled_spring_loaded_relief_valve">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"ball_valve">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    enabled: z.ZodDefault<z.ZodBoolean>;
    clickable: z.ZodDefault<z.ZodBoolean>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"breather_valve">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"butterfly_valve_one">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"butterfly_valve_two">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"check_valve">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"check_valve_with_arrow">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"electric_regulator">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"electric_regulator_motorized">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"four_way_valve">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"gate_valve">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"iso_check_valve">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    enabled: z.ZodDefault<z.ZodBoolean>;
    clickable: z.ZodDefault<z.ZodBoolean>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"manual_valve">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    enabled: z.ZodDefault<z.ZodBoolean>;
    clickable: z.ZodDefault<z.ZodBoolean>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"needle_valve">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"regulator">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"regulator_manual">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    enabled: z.ZodDefault<z.ZodBoolean>;
    clickable: z.ZodDefault<z.ZodBoolean>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"relief_valve">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"solenoid_valve">;
    normallyOpen: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    enabled: z.ZodDefault<z.ZodBoolean>;
    clickable: z.ZodDefault<z.ZodBoolean>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"spring_loaded_relief_valve">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"three_way_valve">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"three_way_ball_valve">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"valve">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"cross_junction">;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"cylinder">;
    dimensions: z.ZodPrefault<z.ZodObject<{
        width: z.ZodNumber;
        height: z.ZodNumber;
    }, z.core.$strip>>;
    borderRadius: z.ZodOptional<z.ZodObject<{
        topLeft: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        topRight: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        bottomLeft: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        bottomRight: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
    }, z.core.$strip>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    backgroundColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"tank">;
    position: z.ZodOptional<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    backgroundColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    dimensions: z.ZodPrefault<z.ZodObject<{
        width: z.ZodNumber;
        height: z.ZodNumber;
    }, z.core.$strip>>;
    borderRadius: z.ZodPrefault<z.ZodObject<{
        topLeft: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        topRight: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        bottomLeft: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        bottomRight: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
    }, z.core.$strip>>;
    fill: z.ZodPrefault<z.ZodObject<{
        stalenessTimeout: z.ZodDefault<z.ZodNumber>;
        stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
        channel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
        rollingAverage: z.ZodOptional<z.ZodInt32>;
        precision: z.ZodDefault<z.ZodNumber>;
        notation: z.ZodDefault<z.ZodEnum<{
            engineering: "engineering";
            scientific: "scientific";
            standard: "standard";
        }>>;
        bounds: z.ZodPrefault<z.ZodObject<{
            lower: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
            upper: z.ZodNumber | z.ZodType<number, unknown, z.core.$ZodTypeInternals<number, unknown>>;
        }, z.core.$strip>>;
        color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
        axisColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
        textColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>>;
        units: z.ZodDefault<z.ZodString>;
        fillHidden: z.ZodDefault<z.ZodBoolean>;
        caretHidden: z.ZodDefault<z.ZodBoolean>;
        scaleHidden: z.ZodDefault<z.ZodBoolean>;
        side: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        caretSide: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
    }, z.core.$strip>>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    variant: z.ZodLiteral<"t_junction">;
}, z.core.$strip>, z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stalenessColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    stateChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    commandChannel: z.ZodOptional<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    control: z.ZodOptional<z.ZodObject<{
        authority: z.ZodOptional<z.ZodInt>;
        hidden: z.ZodDefault<z.ZodBoolean>;
        chipHidden: z.ZodDefault<z.ZodBoolean>;
        indicatorHidden: z.ZodDefault<z.ZodBoolean>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, z.core.$strip>>;
    onClickDelay: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"custom_actuator">;
    specKey: z.ZodString;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    stateOverrides: z.ZodDefault<z.ZodArray<z.ZodObject<{
        key: z.ZodString;
        name: z.ZodString;
        regions: z.ZodDefault<z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            name: z.ZodString;
            selectors: z.ZodDefault<z.ZodArray<z.ZodString>>;
            strokeColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodNumber;
            }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
            }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
            } | [number, number, number] | [number, number, number, number]>>>;
            fillColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodNumber;
            }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
            }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
            } | [number, number, number] | [number, number, number, number]>>>;
        }, z.core.$strip>>>;
    }, z.core.$strip>>>;
}, z.core.$strip>, z.ZodObject<{
    label: z.ZodPrefault<z.ZodObject<{
        label: z.ZodDefault<z.ZodString>;
        level: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: z.ZodDefault<z.ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: z.ZodDefault<z.ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: z.ZodDefault<z.ZodNumber>;
        align: z.ZodDefault<z.ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, z.core.$strip>>;
    orientation: z.ZodDefault<z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: z.ZodDefault<z.ZodNumber>;
    variant: z.ZodLiteral<"custom_static">;
    specKey: z.ZodString;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    } | [number, number, number] | [number, number, number, number]>>>;
    stateOverrides: z.ZodDefault<z.ZodArray<z.ZodObject<{
        key: z.ZodString;
        name: z.ZodString;
        regions: z.ZodDefault<z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            name: z.ZodString;
            selectors: z.ZodDefault<z.ZodArray<z.ZodString>>;
            strokeColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodNumber;
            }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
            }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
            } | [number, number, number] | [number, number, number, number]>>>;
            fillColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodNumber;
            }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
            }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
            } | [number, number, number] | [number, number, number, number]>>>;
        }, z.core.$strip>>>;
    }, z.core.$strip>>>;
}, z.core.$strip>, z.ZodObject<{
    variant: z.ZodLiteral<"group_box">;
    members: z.ZodDefault<z.ZodArray<z.ZodString>>;
    locked: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>], "variant">;
export type Config = schematic.NodeConfig;
export type ConfigOf<V extends Variant> = Extract<Config, {
    variant: V;
}>;
/**
 * Input is the schema input for a variant: its discriminator plus its fields, of which
 * those with a schema default are optional and the rest required.
 */
export type Input<V extends Variant> = {
    variant: V;
} & Omit<Extract<z.input<typeof configZ>, {
    variant: V;
}>, "variant">;
export declare const resolveSpec: (variant: string) => Spec;
/**
 * Builds a fresh config from the schema input. Every unset value comes from the
 * schema, except the label, which names the symbol unless the input sets it.
 * @param input - The variant plus any fields to set on top of the schema defaults.
 * @throws {NotFoundError} if no spec is registered for the variant.
 */
export declare const createConfig: <V extends Variant>(input: Input<V>) => ConfigOf<V>;
export type CustomVariant = typeof CUSTOM_ACTUATOR_VARIANT | typeof CUSTOM_STATIC_VARIANT;
export type CustomConfig = ConfigOf<CustomVariant>;
export declare const CUSTOM_VARIANTS: ReadonlySet<Variant>;
export declare const isCustomVariant: (variant: string | undefined) => variant is CustomVariant;
export declare const isCustomConfig: (config: Config) => config is CustomConfig;
export declare const STATIC_SPECS: readonly Spec[];
//# sourceMappingURL=registry.d.ts.map