import time

import synnax as sy
from synnax.telem.authority import Authority

client = sy.Synnax()

with client.control.acquire(
    name="Press Sequence",
    write_authorities=[255-1],
    write=["press_en_cmd", "press_en_cmd_time"],
    read=["pressure"],
) as auto:
    mark = 100
    while True:
        print("DOG")
        auto["press_en_cmd"] = True
        auto.wait_until(lambda c: c.pressure > mark)
        auto["press_en_cmd"] = False
        print("HELLO")
        mark += 100
        time.sleep(5)



