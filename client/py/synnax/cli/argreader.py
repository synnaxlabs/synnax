import argparse
import os
from enum import Enum
from os import path
from tokenize import Double


class Filetype(Enum):
    CSV = ".csv"
    XLSX = ".xlsx"


class ArgParser:
    def __init__(self):
        self.parser = argparse.ArgumentParser(
            description="CLI for Reading Flat Files to Synnax"
        )
        self.init_parser(self.parser)
        self.args = self.parser.parse_args()

    def init_parser(self, parser):
        parser.add_argument("filepath", type=str, help="filepath of the file to parse")
        parser.add_argument("hostname", type=str, help="IP to synnax node")
        parser.add_argument("port", type=int, default=8080, help="Port of synnax node")
        parser.add_argument(
            "-n",
            "--testname",
            type=str,
            nargs=1,
            metavar="testname",
            help="Name of test",
        )
        parser.add_argument(
            "-t",
            "--timestamp",
            type=str,
            nargs=1,
            metavar="timestamp",
            help="name of timestamp column",
        )

        parser.add_argument(
            "-d",
            "--datarate",
            type=float,
            nargs=1,
            metavar="datarate",
            help="Datarate to use to push to synnax cluster. Will override any timestamp column input.",
        )

        # TAGS:
        parser.add_argument(
            "-f",
            "--force",
            default=False,
            action="store_true",
            # metavar="force",
            help="Create nonexistent channels and push data regardless of timestamp comflicts (DANGEROUS). Defaults to False",
        )
        parser.add_argument(
            "-c",
            "--create",
            default=False,
            action="store_true",
            # metavar="force",
            help="Create nonexistent channels. Defaults to False",
        )
        parser.add_argument(
            "--no_empty",
            default=False,
            action="store_true",
            help="Ignore columns with no data. Defaults to False.",
        )

    def get_filetype(self):
        filename, fileExtension = os.path.splitext(self.args.filepath)
        if fileExtension in [member.value for member in Filetype]:
            return Filetype(fileExtension)
        else:
            print("invalid filetype! must be ")
            for member in Filetype:
                print(member.value + ", ", end="")
            exit(-1)

    def get_filepath(self):
        if path.exists(self.args.filepath):
            return self.args.filepath
        else:
            print("File does not exists")
            exit(-10)

    def get_hostname(self):
        return self.args.hostname

    def get_port(self):
        return self.args.port

    def get_timestamp_col(self):
        if self.args.timestamp != None:
            return self.args.timestamp[0]
        return ""

    def get_datarate(self):
        return self.args.datarate

    def get_flags(self):
        return {
            "force": self.args.force,
            "create": self.args.create,
            "no-empty": self.args.no_empty,
        }
