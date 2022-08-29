from datetime import datetime

import arya
import numpy as np

if __name__ == "__main__":
    client = arya.Client(host="localhost", port=3456)
    ch = client.channel.create(
        name="test",
        data_type=np.float64,
        rate=25,
        node_id=1,
    )
    t0 = datetime.now()
    client.data.write(
        ch.key,
        np.random.rand(1000000),
        start=datetime.now()
    )
    print(datetime.now() - t0)



