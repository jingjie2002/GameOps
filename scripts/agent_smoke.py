import argparse
import json
from pathlib import Path
from urllib.error import URLError
from urllib.request import Request, urlopen


DEFAULT_BASE_URL = "http://127.0.0.1:18090"
REQUIRED_MANIFEST_KEYS = ["id:", "name:", "health:", "commands:", "capabilities:", "agent_tools:"]


def read_json(url: str) -> dict:
    request = Request(url, headers={"Accept": "application/json"})
    with urlopen(request, timeout=3) as response:
        if response.status != 200:
            raise RuntimeError(f"{url} returned HTTP {response.status}")
        return json.loads(response.read().decode("utf-8"))


def validate_manifest() -> None:
    manifest = Path(__file__).resolve().parents[1] / "agent.yaml"
    text = manifest.read_text(encoding="utf-8")
    missing = [key for key in REQUIRED_MANIFEST_KEYS if key not in text]
    if missing:
        raise RuntimeError(f"agent.yaml missing keys: {', '.join(missing)}")


def main() -> int:
    parser = argparse.ArgumentParser(description="GameOps Agent smoke test")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL)
    parser.add_argument("--offline", action="store_true", help="validate agent.yaml only")
    args = parser.parse_args()

    validate_manifest()
    if args.offline:
        print("GameOps agent smoke offline ok")
        return 0

    base_url = args.base_url.rstrip("/")
    health = read_json(f"{base_url}/healthz")
    if health.get("status") != "ok":
        raise RuntimeError(f"unexpected health response: {health}")

    capabilities = read_json(f"{base_url}/api/agent/capabilities")
    project = capabilities.get("project", {})
    if project.get("id") != "gameops":
        raise RuntimeError(f"unexpected capabilities project: {project}")
    agent_tools = capabilities.get("agent_tools", {})
    if agent_tools.get("gameops_analyze_gm_risk") != "POST /api/risk/analyze":
        raise RuntimeError(f"risk analyze tool missing: {agent_tools}")

    for path in ("/api/agent/events", "/api/agent/logs"):
        payload = read_json(f"{base_url}{path}")
        if payload.get("status") != "ok" or payload.get("project") != "gameops":
            raise RuntimeError(f"unexpected {path} response: {payload}")

    print("GameOps agent smoke ok")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except URLError as exc:
        raise SystemExit(f"GameOps service is not reachable: {exc}") from exc
