
exec_context = {
    "dog": 1
}

exec("""
def hello():
    return dog + 1
result = hello()
""", exec_context)
print(exec_context['result'])
