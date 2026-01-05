#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# Time
DAQ_TIME = "daq_time"

# OX
OX_TPC_CMD = "ox_tpc_cmd"
OX_TPC_ACK = "ox_tpc_state"
OX_MPV_CMD = "ox_mpv_cmd"
OX_MPV_ACK = "ox_mpv_state"
OX_PRESS_CMD = "ox_press_cmd"
OX_PRESS_STATE = "ox_press_state"
OX_VENT_CMD = "ox_vent_cmd"
OX_VENT_STATE = "ox_vent_state"
OX_TC_1 = "ox_tc_1"
OX_TC_2 = "ox_tc_2"
OX_PT_1 = "ox_pt_1"
OX_PT_2 = "ox_pt_2"

# FUEL
FUEL_TPC_CMD = "fuel_tpc_cmd"
FUEL_TPC_STATE = "fuel_tpc_state"
FUEL_MPV_CMD = "fuel_mpv_cmd"
FUEL_MPV_STATE = "fuel_mpv_state"
FUEL_PRESS_CMD = "fuel_press_cmd"
FUEL_PRESS_STATE = "fuel_press_state"
FUEL_VENT_CMD = "fuel_vent_cmd"
FUEL_VENT_STATE = "fuel_vent_state"
FUEL_TC_1 = "fuel_tc_1"
FUEL_TC_2 = "fuel_tc_2"
FUEL_PT_1 = "fuel_pt_1"
FUEL_PT_2 = "fuel_pt_2"

# PRESS
PRESS_ISO_CMD = "press_iso_cmd"
PRESS_ISO_STATE = "press_iso_state"
PRESS_PT_1 = "press_pt_1"
PRESS_PT_2 = "press_pt_2"
PRES_TC_1 = "press_tc_1"
PRES_TC_2 = "press_tc_2"

# GAS BOOSTER ISO
GAS_BOOSTER_ISO_CMD = "gas_booster_iso_cmd"
GAS_BOOSTER_ISO_STATE = "gas_booster_iso_state"

SUPPLY_PT = "supply_pt"
PNEUMATICS_PT = "pneumatics_pt"

VALVES = {
    OX_TPC_CMD: OX_TPC_ACK,
    OX_MPV_CMD: OX_MPV_ACK,
    OX_PRESS_CMD: OX_PRESS_STATE,
    OX_VENT_CMD: OX_VENT_STATE,
    FUEL_TPC_CMD: FUEL_TPC_STATE,
    FUEL_MPV_CMD: FUEL_MPV_STATE,
    FUEL_PRESS_CMD: FUEL_PRESS_STATE,
    FUEL_VENT_CMD: FUEL_VENT_STATE,
    PRESS_ISO_CMD: PRESS_ISO_STATE,
    GAS_BOOSTER_ISO_CMD: GAS_BOOSTER_ISO_STATE,
}

PTS = (
    OX_PT_1,
    OX_PT_2,
    FUEL_PT_1,
    FUEL_PT_2,
    PRESS_PT_1,
    PRESS_PT_2,
    SUPPLY_PT,
    PNEUMATICS_PT,
)

SENSORS = (OX_TC_1, OX_TC_2, FUEL_TC_1, FUEL_TC_2, PRES_TC_1, PRES_TC_2, *PTS)
