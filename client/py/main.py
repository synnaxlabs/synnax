import time
from datetime import datetime

from arya import Client, telem

import numpy as np

if __name__ == "__main__":
    client = Client(host="localhost", port=3457)

    ch = client.channel.create(
        name="temperatureSensor1",
        rate=25 * telem.KHZ,
        data_type=np.float64,
        node_id=1,
    )
    time.sleep(2)

    random_sensor_data = np.random.rand(325000)
    print(random_sensor_data)

    start = datetime.now()
    t0 = telem.now()

    client.data.write(ch.key, random_sensor_data, t0)

    res_data = client.data.read(ch.key, telem.TimeRange(
        t0 - 3 * telem.HOUR,
        t0 + 3 * telem.HOUR,
    ))

    np.array_equal(random_sensor_data, res_data)
    print(datetime.now() - start)



