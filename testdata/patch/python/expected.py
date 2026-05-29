def handle_resources_read(uri: str):
    resource = store.get(uri)
    if resource is None:
        return {"error": {"code": -32602, "message": "resource not found"}}
    return {"result": resource}


def read_resource(uri: str):
    if uri not in store:
        return {
            "error": {
                "code": -32602,
                "message": "not found",
            }
        }
    return {"result": store[uri]}
