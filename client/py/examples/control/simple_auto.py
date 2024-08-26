import synnax as sy

client = sy.Synnax()

with client.control.acquire(
    name="Scheduled Commands",
    read=["pressure_1", "temperature_1", "valve_1_state", "valve_2_state"],
    write=["valve_1_cmd", "valve_2_cmd", "valve_3_cmd", "valve_4_cmd"],
) as ctrl:
    # Set initial valve states
    ctrl.set(
        {
            "valve_1_cmd": False,
            "valve_2_cmd": True,
        }
    )

    # Wait until pressure_1 is less than 100 psi
    ctrl.wait_until(lambda auto: auto["pressure_1"] < 100)
    ctrl.sleep(1)

    # Schedule valve commands on the driver side exactly 30 milliseconds apart.
    ctrl.schedule(
        sy.ScheduledCommand(at=0, values={"valve_3_cmd": True}),
        sy.ScheduledCommand(
            at=sy.TimeSpan.MILLISECOND * 30, values={"valve_2_cmd": True}
        ),
    )

    # Wait until the second valve has been opened before proceeding
    ctrl.wait_until(lambda auto: auto["valve_2_ack"] == True)
