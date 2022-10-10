import numpy as np
import uuid
import synnax
from datetime import datetime

from synnax.channel import retrieve


class Converter:
    def __init__(
        self,
        filereader,
        flags,
        hostname="localhost",
        testname="",
        port=8080,
        timestampCol="gse.timestamp (ul)",
        datarate=0,
    ):
        self.filereader = filereader
        self.hostname = hostname
        self.port = port
        self.testUUID = uuid.uuid4()  # TODO
        self.datarate = datarate
        self.flags = flags

        if timestampCol != "" and self.datarate == 0:
            self.datarate = self.datect_data_rate(timestampCol)
        elif self.flags.get("force"):
            print("(FORCE) Will use existing datarates")
        else:
            print("Failed to detect datarate")
            exit(10)

        self.client = synnax.Synnax(host=self.hostname, port=self.port)

    def set_target(self, hostname, port):
        self.hostname = hostname
        self.port = port
        self.client = synnax.Synnax(host=hostname, port=port)

    def set_test_name(self, name):
        self.tags.name = name

    def set_flag(self, key, value):
        self.tags.flags.update({key, value})

    def get_tags(self):
        return self.tags

    def get_cols(self):
        pass

    # Gets the datarate from the timestamp column
    # TODO
    def datect_data_rate(self, timestampCol):
        cols = self.filereader.get_headers()

        print(timestampCol)

        if not timestampCol in cols:
            print("Invalid Timestamp Column Name")
            exit(-11)

        tsc = self.filereader.get_cols_sample(timestampCol)

        # print("test")
        # print(tsc)

        diffs = []
        last = tsc.iat[0, 0]
        for ind, row in tsc.itterrows():
            print("Here")
            print(row)
            diffs.append(row[1] - last)
            last = row[1]

        print(diffs)

    # Chunk is part of a pandas dataframe
    def parseChunk(self, chunk):

        if self.flags.get("no-empty"):
            chunk.dropna(how="all", axis=1, inplace=True)

        channels = retrieveChannels(chunk.columns.tolist(), self.client)

        # channels = self.client.channel.retrieve_by_name(chunk.columns.tolist())

        channel_names = [ch.name for ch in channels]
        for ch in chunk.columns.tolist():
            if not ch in channel_names:
                print("Channel Not Found")
                # Todo --force tag or smthng to force push data
                # Todo --create tag or smthng would be better
                if self.flags.get("create") or self.flags.get("force"):
                    print("Creating channel " + ch)
                    if self.datarate == 0 and self.flags.get("force"):
                        # Can only be here if force is true
                        # Only adding flag check to be extra safe
                        self.datarate = channels[0].rate
                        print("FORCE: Using datarate " + self.datarate)

                    print(chunk[ch].describe())
                    print(chunk[ch].dtypes)
                    self.client.channel.create(
                        name=ch, rate=self.datarate, data_type=np.float64
                    )
                else:
                    exit()
        if self.flags.get("create") or self.flags.get("force"):
            channels = retrieveChannels(chunk.columns.tolist(), self.client)

        if not self.flags.get("force"):
            channel_datarates = [ch.rate for ch in channels]
            for dr in channel_datarates:
                if dr != self.datarate * synnax.HZ:
                    print("Datarates do not match")
                    print(dr)
                    print(self.datarate)
                    exit()

        start = datetime.now()

        for ch in channels:
            ch.write(start, chunk[ch.name].to_numpy(dtype=ch.data_type))

        print("wrote data")

    def parse(self):
        self.filereader.push_data_chunks(self)


# external to class just a helper function
def retrieveChannels(cols, client):
    channels = []
    for col in cols:
        ch = client.channel.retrieve_by_name([col])
        if len(ch) != 0:
            channels.append(ch[0])

    return channels
