
import synnax as sy

client = sy.Synnax()

with client.control.acquire(
    name="Press Sequence",
    write_authorities=[sy.Authority.ABSOLUTE - 1],
    write=["press_en_cmd", "press_en_cmd_time"],
    read=["pressure"],
) as auto:




