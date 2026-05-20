import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request


ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
ADDR = os.environ.get("GAMEOPS_RISK_DEMO_ADDR", "127.0.0.1:18091")
BASE_URL = f"http://{ADDR}"


def request(method, path, token=None, payload=None, expected=(200, 201)):
    data = None
    headers = {}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(BASE_URL + path, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=3) as resp:
            body = resp.read().decode("utf-8")
            if resp.status not in expected:
                raise RuntimeError(f"{method} {path} expected {expected}, got {resp.status}: {body}")
            return json.loads(body) if body else None
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        if exc.code in expected:
            return json.loads(body) if body else None
        raise RuntimeError(f"{method} {path} failed {exc.code}: {body}") from exc


def wait_ready():
    for _ in range(40):
        try:
            request("GET", "/healthz")
            return
        except Exception:
            time.sleep(0.25)
    raise RuntimeError("GameOps did not become ready")


def main():
    tmp_dir = os.path.join(ROOT, "tmp")
    os.makedirs(tmp_dir, exist_ok=True)
    exe = os.path.join(tmp_dir, "gameops-server.exe" if os.name == "nt" else "gameops-server")

    env = os.environ.copy()
    env["GOCACHE"] = env.get("GOCACHE", os.path.join(ROOT, ".gocache"))
    env["GAMEOPS_ADDR"] = ADDR
    env["GAMEOPS_RISK_AI_PROVIDER"] = "mock-ai"

    subprocess.run(["go", "build", "-o", exe, "./cmd/server"], cwd=ROOT, env=env, check=True)
    proc = subprocess.Popen([exe], cwd=ROOT, env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        wait_ready()

        login = request("POST", "/api/admin/login", payload={"username": "admin", "password": "admin_demo"})
        token = login["token"]
        print(f"admin_login role={login['role']}")

        players = request("POST", "/api/players/seed", token=token, payload={})
        print(f"seed_players count={len(players)}")

        for index in range(4):
            player_id = f"player_100{index % 3 + 1}"
            mail = request("POST", "/api/mails", token=token, payload={
                "player_id": player_id,
                "title": f"Emergency compensation {index + 1}",
                "body": "Risk demo reward mail",
                "gold": 1200 + index * 300,
                "items": ["ticket"],
            })
            print(f"create_risk_mail {mail['mail_id']} player={player_id} gold={mail['gold']}")

        batch = request("POST", "/api/cdk/batches", token=token, payload={
            "name": "risk demo batch",
            "gold": 200,
            "items": ["gem"],
            "count": 3,
            "max_uses_per_code": 1,
            "expires_in_seconds": 3600,
        })
        print(f"create_cdk_batch {batch['batch_id']} codes={len(batch['codes'])}")

        for code in batch["codes"]:
            redeemed = request("POST", f"/api/cdk/{code}/redeem", payload={"player_id": "player_1002"})
            print(f"redeem_cdk {redeemed['code']} player={redeemed['player_id']}")

        for index, value in enumerate(["true", "false", "true"], start=1):
            cfg = request("PUT", "/api/ops-configs/ranked_maintenance", token=token, payload={
                "config_value": value,
                "description": f"risk demo maintenance toggle {index}",
            })
            print(f"update_config {cfg['config_key']}={cfg['config_value']}")

        for index in range(3):
            banned = request("POST", "/api/players/player_1003/ban", token=token, payload={
                "reason": f"risk_demo_{index + 1}",
                "banned_seconds": 60,
            })
            print(f"ban_player {banned['player_id']} reason={banned['ban_reason']}")

        report = request("POST", "/api/risk/analyze", token=token, payload={"use_ai": True})
        print(f"risk_project={report['project_name']}")
        print(f"risk_level={report['risk_level']} score={report['score']} ai_provider={report['ai_provider']}")
        print(f"summary={report['summary']}")
        print("findings:")
        for finding in report["findings"]:
            print(f"- {finding['severity']} {finding['type']}: {finding['reason']}")
            for evidence in finding["evidence"][:2]:
                print(f"  evidence {evidence['audit_id']}: {evidence['message']}")
            print(f"  suggestion: {finding['suggestion']}")
        print("GameOps AI risk demo completed")
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        raise
