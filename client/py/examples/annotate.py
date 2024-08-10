import synnax as sy
import uuid

client = sy.Synnax()

# Retrieve Bang Bang Sim 
rng = client.ranges.retrieve(uuid.UUID("93623262-8869-4a38-89c3-7d0472884e69"))
rng.meta_data[str(uuid.uuid4())] = "cat"
    