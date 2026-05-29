async function handleResourcesRead(uri) {
  const resource = store.get(uri);
  if (!resource) {
    return { error: { code: -32002, message: "resource not found" } };
  }
  return { result: resource };
}

// multi-line style
async function readResource(uri) {
  if (!store.has(uri)) {
    return {
      error: {
        code: -32002,
        message: "not found",
      },
    };
  }
  return { result: store.get(uri) };
}
