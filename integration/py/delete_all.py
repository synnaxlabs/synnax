import sys

import synnax as sy

client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


def main():
    channels = client.channels.retrieve([".*"])
    client.channels.delete([channel.key for channel in channels if not channel.internal])


if __name__ == "__main__":
    main()
