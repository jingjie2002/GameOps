import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request


ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
ADDR = "127.0.0.1:18090"
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

    subprocess.run(["go", "build", "-o", exe, "./cmd/server"], cwd=ROOT, env=env, check=True)
    proc = subprocess.Popen([exe], cwd=ROOT, env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        wait_ready()

        login = request("POST", "/api/admin/login", payload={"username": "admin", "password": "admin_demo"})
        token = login["token"]
        print(f"admin_login role={login['role']}")

        players = request("POST", "/api/players/seed", token=token, payload={})
        print(f"seed_players count={len(players)}")

        banned = request("POST", "/api/players/player_1003/ban", token=token, payload={
            "reason": "abuse_report",
            "banned_seconds": 60,
        })
        print(f"ban_player {banned['player_id']} status={banned['status']}")

        unbanned = request("POST", "/api/players/player_1003/unban", token=token, payload={})
        print(f"unban_player {unbanned['player_id']} status={unbanned['status']}")

        cfg = request("PUT", "/api/ops-configs/ranked_maintenance", token=token, payload={
            "config_value": "true",
            "description": "ranked queue closed for maintenance",
        })
        print(f"update_config {cfg['config_key']}={cfg['config_value']}")

        state = request("GET", "/api/public/ops-state")
        print(f"public_ops_state ranked_maintenance={state['ranked_maintenance']}")

        mail = request("POST", "/api/mails", token=token, payload={
            "player_id": "player_1001",
            "title": "SS25 ranked reward",
            "body": "Season compensation",
            "gold": 500,
            "items": ["skin_trial"],
        })
        print(f"create_mail {mail['mail_id']}")

        claimed = request("POST", f"/api/players/player_1001/mails/{mail['mail_id']}/claim", payload={})
        print(f"claim_mail {claimed['mail_id']} status={claimed['status']}")

        duplicate_claim = request("POST", f"/api/players/player_1001/mails/{mail['mail_id']}/claim", payload={}, expected=(409,))
        print(f"duplicate_claim blocked={duplicate_claim['error']}")

        batch = request("POST", "/api/cdk/batches", token=token, payload={
            "name": "launch gift",
            "gold": 300,
            "items": ["ticket"],
            "count": 1,
            "max_uses_per_code": 1,
            "expires_in_seconds": 3600,
        })
        code = batch["codes"][0]
        print(f"create_cdk_batch {batch['batch_id']} code={code}")

        redeemed = request("POST", f"/api/cdk/{code}/redeem", payload={"player_id": "player_1002"})
        print(f"redeem_cdk {redeemed['code']} player={redeemed['player_id']}")

        duplicate_redeem = request("POST", f"/api/cdk/{code}/redeem", payload={"player_id": "player_1002"}, expected=(409,))
        print(f"duplicate_redeem blocked={duplicate_redeem['error']}")

        event = request("POST", "/api/events", payload={
            "type": "reward_claim",
            "player_id": "player_1001",
            "payload": {"source": "demo_flow"},
        })
        print(f"event_ingested {event['event_id']} type={event['type']}")

        audits = request("GET", "/api/audit-logs", token=token)
        print(f"audit_logs count={len(audits)}")

        metrics = urllib.request.urlopen(BASE_URL + "/metrics", timeout=3).read().decode("utf-8")
        if "gameops_requests_total" not in metrics or "gameops_cdk_redeems_total" not in metrics:
            raise RuntimeError("metrics missing expected counters")
        print("metrics include gameops_requests_total and gameops_cdk_redeems_total")
        print("GameOps demo completed")
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
