-- EMBEDDED SEQUENCE BENCHMARK CLICK CHECK SEQUENCE
-- Written by Ben Richards
-- Reference: https://docs.synnaxlabs.com/reference/control/embedded/reference


-- FUNCTION DEFINITIONS
function verifyLinearity(run_mode_timings)
    local previous_time = -1
    for i, v in ipairs(run_mode_timings) do
        if v[2] > previous_time then
            previous_time = v[2]
        else
            set("debug_channel", previous_time)
            return false
        end
    end
end


-- GLOBAL VARIABLE DEFINITIONS
if iteration == 1 then

    -- SET authorities for different modes
    ABORT_AUTHORITY = 250
    ESTOP_AUTHORITY = 251
    NOMINAL_AUTHORITY = 100
    ELEVATED_AUTHORITY = 200
    PASSIVE_AUTHORITY = 1

    -- NOMINAL AUTHORITY SETTING
    set_authority("PV_FU_01_cmd", NOMINAL_AUTHORITY)
    set_authority("STATE", NOMINAL_AUTHORITY)
    set_authority("auto_trigger", ELEVATED_AUTHORITY)
    set_authority("countdown", NOMINAL_AUTHORITY)
    set_authority("debug_channel", ELEVATED_AUTHORITY)
    set_authority("start_time", ELEVATED_AUTHORITY)

    -- SET integers for different modes
    HOLD_MODE = 0
    RUN_MODE = 1
    ABORT_MODE = 2
    ESTOP_MODE = 3

    -- SET STATE TO HOLD ON STARTUP ALWAYS
    set("STATE", HOLD_MODE)
    set("auto_trigger", 0)
    set("debug_channel", iteration)

    set_authority("auto_trigger", PASSIVE_AUTHORITY)

-- AUTOSEQUENCE TIMING DEFINITIONS
    COUNTDOWN_LENGTH = 5
    RUN_MODE_STEPS = {
        {"PV_FU_01_cmd", 0, true},
        {"PV_FU_01_cmd", 5, false},
        {"STATE", 10, HOLD_MODE}
    }

    --if verifyLinearity(RUN_MODE_STEPS) == false then
    --    set_authority("auto_trigger", ABORT_AUTHORITY)
    --    set("debug_channel", 999)
    --end

    -- dont edit this please i beg of you
    RUN_MODE_STEP = 1
end



-- BEGIN STATE MACHINE
if STATE == HOLD_MODE then
    -- Do nothing until told otherwise

    if auto_trigger == 1 then
        set("STATE", RUN_MODE)
        set("start_time", elapsed_time)
    end
end

if STATE == ABORT_MODE then
    -- INPUT ABORT SEQUENCE HERE
    set("debug_channel_str", "ABORT_MODE")


    -- set STATE to hold
    set("STATE", HOLD_MODE)
end

if STATE == ESTOP_MODE then
    -- INPUT E-STOP SEQUENCE HERE
    set("debug_channel_str", "ESTOP_MODE")


    -- set STATE to hold
    set("STATE", HOLD_MODE)
end

if STATE == RUN_MODE then
    -- INPUT ABORT MODES HERE
    -- if pressure_transducer_that_shouldn't_be_above_300 > 300 then
    --      set("STATE", ABORT_MODE)
    --      return
    -- end
    set("debug_channel_str", "RUN_MODE")

    -- INPUT RUN SEQUENCE HERE
    local autosequence_time = (elapsed_time - start_time) - COUNTDOWN_LENGTH
    set("countdown", autosequence_time)

    -- Index through all and get time

    -- If the autosequnce_time is greater than the step time.
    if autosequence_time > RUN_MODE_STEPS[RUN_MODE_STEP][2] then
        set(RUN_MODE_STEPS[RUN_MODE_STEP][1], RUN_MODE_STEPS[RUN_MODE_STEP][3])


        -- Now i must iterate and see if there are any other channels that actuate at this time.

        RUN_MODE_STEP = RUN_MODE_STEP + 1
    end

end
