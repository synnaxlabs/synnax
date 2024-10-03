import csv
import pandas as pd
import nptdms
from flask import Flask, request
from flask_cors import CORS
import numpy as np
from typing import Protocol
from .overload import overload_comparison_operators
from pathlib import Path

app = Flask(__name__)
CORS(app, resources={r"*": {"origins": "*"}})


class Channel(Protocol):
    ...

    def __array__(self) -> np.array:
        ...


class Runtime:
    values: dict[str, Channel]

    def __init__(self):
        self.values = dict()


runtime = Runtime()


class ArrayChannel(Channel):
    def __init__(self, data: np.array):
        self.data = data

    def __array__(self) -> np.array:
        return self.data


class CalculatedChannel(Channel):
    def __init__(self, name: str, statement: str):
        self.name = name
        self.statement = statement

    def __array__(self) -> np.array:
        ctx = {
            "np": np,
            **{
                col: ArrayChannel(np.array(runtime.values[col])) for col in
                runtime.values if col != self.name
            },
        }
        exec(self.statement, ctx)
        return np.array(ctx['result'])


Channel = overload_comparison_operators(Channel, "__array__")

@app.post('/api/v1/upload')
def upload():
    data = request.get_json()
    if data is None:
        return 'No data received', 400
    path = data.get('path')
    if path is None:
        return 'No path received', 400
    p = Path(path)
    if p.suffix == '.tdms':
        df = read_tdms_channels(path)
    elif p.suffix == '.csv':
        df = read_csv_channels(path)
    else:
        return 'Unsupported file format', 400
    for channel in df:
        runtime.values[channel] = df[channel]
    return 'Data uploaded', 200


def read_tdms_channels(path: str) -> dict[str, Channel]:
    tdms_file = nptdms.TdmsFile.read(path)
    return {channel: ArrayChannel(np.array(tdms_file[channel])) for channel in
            tdms_file.groups()}


def read_csv_channels(path: str) -> dict[str, Channel]:
    df = pd.read_csv(path)
    return {col: ArrayChannel(np.array(df[col])) for col in df.columns}


@app.post('/api/v1/value')
def value():
    data = request.get_json()
    if data is None:
        return 'No data received', 400
    channel = data.get('channel')
    if channel is None:
        return 'No channel received', 400
    return {"value": [int(v) for v in list(runtime.values[channel].__array__())]}


@app.post('/api/v1/create_calculated')
def create_calculated():
    statement = request.get_json().get('statement')
    if statement is None:
        return 'No statement received', 400
    name = request.get_json().get('name')
    if name is None:
        return 'No name received', 400
    runtime.values[name] = CalculatedChannel(name, statement)
    return "", 200


@app.post('/api/v1/delete_calculated')
def delete_calculated():
    del runtime.values['result']
    return "", 200


@app.post('/api/v1/list')
def list_channels():
    return {"channels": list(runtime.values.keys())}
