import numpy as np
import uuid


class Converter:
    def __init__(
        self,
        df,
        hostname="localhost",
        testname="",
        port=8080,
        timestampCol="gse.timestamp (ul)",
    ):
        self.df = df
        self.hostname = hostname
        self.port = port
        self.tags = {"name": testname, "uuid": uuid.uuid4(), "flags": {}}

        if timestampCol != "":
            self.datarate = self.detectDataRate(self, timestampCol)
        else:
            self.datarate = None

    def setTarget(self, hostname, port):
        self.hostname = hostname
        self.port = port

    def setTestName(self, name):
        self.tags.name = name

    def setFlag(self, key, value):
        self.tags.flags.update({key, value})

    def getTags(self):
        return self.tags

    def getDFCols(self):
        pass

    # Gets the datarate from the timestamp column
    def detectDataRate(self):
        pass

    def pushData(self):
        if self.datarate == None:
            print("Need to set a datarate!")
            return
