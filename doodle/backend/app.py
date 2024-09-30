import csv
import pandas as pd
import nptdms
from flask import Flask, request

app = Flask(__name__)

global df


@app.post('/api/v1/tdms/upload')
def upload_tdms():
    global df
    data = request.get_json()
    if data is None:
        return 'No data received', 400
    path = data.get('path')
    if path is None:
        return 'No path received', 400
    tdms_file = nptdms.TdmsFile.read(path)
    df = tdms_file.as_dataframe()


@app.post('/api/v1/csv/upload')
def upload_csv():
    global df
    data = request.get_json()
    if data is None:
        return 'No data received', 400
    path = data.get('path')
    if path is None:
        return 'No path received', 400
    df = pd.read_csv(path)
    return {"columns": df.columns.tolist()}


@app.post('/api/v1/exec')
def execute():
    global df
    exec_context = {
        # Convert all columns to keys with series as their values
        **{col: df[col] for col in df.columns},
    }
    statement = request.get_json().get('statement')
    if statement is None:
        return 'No statement received', 400
    exec(statement, exec_context)
    print(exec_context)
    return {"result": exec_context['result']}
