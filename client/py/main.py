from synnax import Synnax, telem

# client = Synnax(
#     host="161.35.124.196",
#     port=80,
#     username="synnax",
#     password="seldon",
# )

client = Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
)

ch = client.channel.retrieve(name="ec.pressure[7] (psi)")
print(ch.key)

data = ch.read(0, telem.TIME_STAMP_MAX)

print(data)
