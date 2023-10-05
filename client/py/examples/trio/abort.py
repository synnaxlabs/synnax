import synnax as sy
import time
from synnax.telem.authority import Authority

client = sy.Synnax()

with client.control.acquire(
    name="Abort Sequence",
    write_authorities=[0],
    write=[
        "press_en_cmd",
        "press_en_cmd_time",
        "vent_en_cmd",
        "vent_en_cmd_time",
    ],
    read=["pressure"],
) as auto:
    auto.wait_until(lambda c: c.pressure > 1000)
    auto.authorize("press_en_cmd", Authority.ABSOLUTE)
    auto.authorize("vent_en_cmd", Authority.ABSOLUTE)
    auto["press_en_cmd"] = False
    auto["vent_en_cmd"] = True
    time.sleep(1000)
