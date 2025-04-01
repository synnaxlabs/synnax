import synnax as sy

client = sy.Synnax()

task = client.hardware.tasks.retrieve(name="ni scanner")

client.hardware.tasks.delete(task.key)