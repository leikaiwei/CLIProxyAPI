import json
import urllib.error
import urllib.request

REGISTRY = "https://ghcr.io"
PACKAGE = "leikaiwei/cliproxyapi"
ACCEPT = ",".join(
    (
        "application/vnd.oci.image.index.v1+json",
        "application/vnd.docker.distribution.manifest.list.v2+json",
        "application/vnd.oci.image.manifest.v1+json",
        "application/vnd.docker.distribution.manifest.v2+json",
    )
)


def read(url, token=None):
    headers = {"Accept": ACCEPT}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    with urllib.request.urlopen(urllib.request.Request(url, headers=headers), timeout=30) as response:
        return json.load(response)


def published_revision():
    try:
        token = read(f"{REGISTRY}/token?service=ghcr.io&scope=repository:{PACKAGE}:pull")["token"]
        manifest = read(f"{REGISTRY}/v2/{PACKAGE}/manifests/stable", token)
    except urllib.error.HTTPError as error:
        if error.code in (401, 403, 404):
            return ""
        raise
    descriptor = next(item for item in manifest["manifests"] if item["platform"]["architecture"] == "amd64")
    image = read(f"{REGISTRY}/v2/{PACKAGE}/manifests/{descriptor['digest']}", token)
    config = read(f"{REGISTRY}/v2/{PACKAGE}/blobs/{image['config']['digest']}", token)
    return config["config"].get("Labels", {}).get("org.opencontainers.image.revision", "")


if __name__ == "__main__":
    print(published_revision())
