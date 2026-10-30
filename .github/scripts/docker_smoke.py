"""Exercise a local image using disposable Docker resources, then clean up.

The workflow sets a local DOCKER_HOST and an empty DOCKER_CONFIG. Evidence is
written separately from temporary credentials. --cleanup is idempotent and also
runs as an always() workflow step in case the smoke step is interrupted.
"""

import argparse
import datetime
import gzip
import http.cookiejar
import json
import math
from pathlib import Path
import re
import secrets
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid


LABEL = "io.koito.disposable-smoke"
FIELDS = {
    "listen_count", "track_count", "album_count", "artist_count",
    "minutes_listened", "days_active", "longest_streak", "avg_daily_plays",
    "tracks_per_artist", "albums_per_artist",
}


class RejectRedirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def docker(*args, check=True):
    result = subprocess.run(
        ["docker", *args], capture_output=True, text=True, timeout=30,
    )
    if check and result.returncode:
        raise RuntimeError(f"docker {' '.join(args[:2])}: {result.stderr.strip()}")
    return result


def inspect(kind, name):
    result = docker(kind, "inspect", name, check=False)
    if result.returncode:
        if "no such" in result.stderr.lower():
            return None
        raise RuntimeError(result.stderr.strip())
    return json.loads(result.stdout)[0]


def cleanup(work):
    state_file = work / "resources.json"
    if not state_file.exists():
        return
    state = json.loads(state_file.read_text())
    for kind, key in (("container", "container"), ("volume", "volume")):
        info = inspect(kind, state[key])
        if info is None:
            continue
        labels = info["Config"].get("Labels", {}) if kind == "container" else info.get("Labels", {})
        if (labels or {}).get(LABEL) != state["id"]:
            raise RuntimeError(f"Refusing to remove {kind}: ownership label mismatch")
        if kind == "container":
            docker("stop", "--time", "10", state[key])
        docker(kind, "rm", state[key])
    (work / "test.env").unlink(missing_ok=True)


def smoke(image, architecture, work):
    evidence = work / "evidence"
    evidence.mkdir(parents=True, exist_ok=True)
    state_file = work / "resources.json"
    if state_file.exists():
        raise RuntimeError("Use a fresh work directory for every smoke test")
    token = uuid.uuid4().hex
    state = {"id": token, "container": f"koito-smoke-{token}", "volume": f"koito-smoke-{token}-data"}
    state_file.write_text(json.dumps(state))
    password = secrets.token_urlsafe(24)
    username = "disposable-smoke"
    api_key = ""
    report = {"checks": [], "resources": [], "names": state}
    failure = None

    def redact(text):
        text = text.replace(password, "[redacted-password]")
        return text.replace(api_key, "[redacted-api-key]") if api_key else text

    def check(name, condition, detail=None):
        report["checks"].append({"name": name, "result": "PASS" if condition else "FAIL", "detail": detail})
        if not condition:
            raise AssertionError(name)

    # Disable proxy inheritance and keep authenticated and anonymous clients separate.
    jar = http.cookiejar.CookieJar()
    public = urllib.request.build_opener(urllib.request.ProxyHandler({}), RejectRedirects())
    authenticated = urllib.request.build_opener(
        urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(jar), RejectRedirects(),
    )
    base = ""

    def request(name, path, body=None, headers=None, session=False):
        headers = dict(headers or {})
        data = None
        if body is not None:
            headers["Content-Type"] = "application/json"
            data = json.dumps(body).encode()
        url = urllib.parse.urljoin(base + "/", path)
        if urllib.parse.urlparse(url).netloc != urllib.parse.urlparse(base).netloc:
            raise RuntimeError("Refusing an HTTP request outside the test listener")
        req = urllib.request.Request(url, data=data, headers=headers)
        started = time.monotonic()
        try:
            response = (authenticated if session else public).open(req, timeout=5)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            code, hdr, body = response.status, response.headers, response.read()
        report.setdefault("http", []).append({"name": name, "status": code, "seconds": time.monotonic() - started, "bytes": len(body)})
        if 300 <= code < 400:
            raise RuntimeError(f"Unexpected HTTP redirect for {name} (status {code})")
        if hdr.get("Content-Encoding") == "gzip":
            body = gzip.decompress(body)
        return code, hdr, body

    def stats(name, expected):
        code, hdr, body = request(name, "/apis/web/v1/stats?period=all_time&tz=UTC", session=True)
        values = json.loads(body)
        check(name, code == 200 and hdr.get_content_type() == "application/json"
              and set(values) == FIELDS
              and all(type(v) in (int, float) and math.isfinite(v) for v in values.values())
              and values == expected, values)

    try:
        metadata = inspect("image", image)
        check("image architecture", metadata is not None and metadata["Os"] == "linux" and metadata["Architecture"] == architecture)
        (evidence / "image.json").write_text(json.dumps({k: metadata[k] for k in ("Id", "Os", "Architecture", "Size")}, indent=2))
        check("fresh resource names", inspect("container", state["container"]) is None and inspect("volume", state["volume"]) is None)
        docker("volume", "create", "--label", f"{LABEL}={token}", state["volume"])
        env = {
            "KOITO_CONFIG_DIR": "/etc/koito", "KOITO_BIND_ADDR": "0.0.0.0", "KOITO_LISTEN_PORT": "4110",
            "KOITO_DEFAULT_USERNAME": username, "KOITO_DEFAULT_PASSWORD": password,
            "KOITO_LOGIN_GATE": "true", "KOITO_FORCE_TZ": "UTC", "TZ": "UTC",
            "KOITO_SKIP_IMPORT": "true", "KOITO_ENABLE_LBZ_RELAY": "false",
            "KOITO_DISABLE_MUSICBRAINZ": "true", "KOITO_DISABLE_DEEZER": "true",
            "KOITO_DISABLE_COVER_ART_ARCHIVE": "true", "KOITO_ENABLE_FULL_IMAGE_CACHE": "false",
            "KOITO_ENABLE_STRUCTURED_LOGGING": "true", "KOITO_LOG_LEVEL": "info",
        }
        env_file = work / "test.env"
        env_file.touch(mode=0o600)
        env_file.write_text("".join(f"{k}={v}\n" for k, v in env.items()))
        docker("run", "--detach", "--pull=never", "--platform", f"linux/{architecture}",
               "--name", state["container"], "--label", f"{LABEL}={token}", "--restart=no",
               "--publish", "127.0.0.1::4110/tcp", "--mount", f"type=volume,source={state['volume']},target=/etc/koito",
               "--env-file", str(env_file), image)
        info = inspect("container", state["container"])
        check("container starts", info["State"]["Running"], info["State"])
        ports = info["NetworkSettings"]["Ports"].get("4110/tcp")
        mounts = info["Mounts"]
        check("loopback and disposable volume", bool(ports) and all(p["HostIp"] == "127.0.0.1" for p in ports)
              and len(mounts) == 1 and mounts[0]["Type"] == "volume"
              and mounts[0]["Name"] == state["volume"] and mounts[0]["Destination"] == "/etc/koito")
        base = "http://127.0.0.1:" + ports[0]["HostPort"]
        ready = False
        deadline = time.monotonic() + 60
        while time.monotonic() < deadline:
            if not inspect("container", state["container"])["State"]["Running"]:
                break
            try:
                ready = request("readiness", "/apis/web/v1/health")[0] == 200
                if ready:
                    break
            except (OSError, urllib.error.URLError):
                pass
            time.sleep(0.5)
        check("HTTP readiness", ready)
        linked = docker("exec", state["container"], "ldd", "/app/app", check=False)
        (evidence / "ldd.log").write_text(linked.stdout + linked.stderr)
        check("CGO and libvips linkage", linked.returncode == 0 and "not found" not in linked.stdout + linked.stderr and "libvips.so" in linked.stdout)
        code, hdr, html = request("web", "/")
        check("web HTML", code == 200 and hdr.get_content_type() == "text/html" and b"<html" in html.lower())
        assets = re.findall(r'''(?:src|href)=["']([^"']+\.js(?:\?[^"']*)?)["']''', html.decode())
        check("JavaScript reference", bool(assets))
        code, hdr, body = request("JavaScript", assets[0], headers={"Accept-Encoding": "gzip"})
        check("web JavaScript", code == 200 and bool(body) and hdr.get_content_type() != "text/html")
        check("login gate", request("anonymous stats", "/apis/web/v1/stats?period=all_time")[0] == 401)
        check("invalid password rejected", request("bad login", "/apis/web/v1/login", {"username": username, "password": "wrong"})[0] == 401)
        check("login", request("login", "/apis/web/v1/login", {"username": username, "password": password}, session=True)[0] == 204 and any(c.name == "koito_session" for c in jar))
        code, _, body = request("session", "/apis/web/v1/user", session=True)
        check("authenticated session", code == 200 and json.loads(body).get("username") == username)
        stats("empty statistics", dict.fromkeys(FIELDS, 0))
        code, _, body = request("API key", "/apis/web/v1/user/apikeys", session=True)
        keys = json.loads(body)
        check("test API key", code == 200 and bool(keys) and bool(keys[0].get("key")))
        api_key = keys[0]["key"]
        yesterday = datetime.datetime.now(datetime.timezone.utc).date() - datetime.timedelta(days=1)
        at = int(datetime.datetime.combine(yesterday, datetime.time(12), tzinfo=datetime.timezone.utc).timestamp())
        listen = {"listen_type": "single", "payload": [{"listened_at": at, "track_metadata": {
            "artist_name": "Disposable Smoke Artist", "track_name": "Disposable Smoke Track",
            "release_name": "Disposable Smoke Album", "additional_info": {"duration": 180},
        }}]}
        check("first-listen ingestion", request("ingestion", "/apis/listenbrainz/1/submit-listens", listen, {"Authorization": "Token " + api_key})[0] == 200)
        stats("populated statistics", {k: 3 if k == "minutes_listened" else 1 for k in FIELDS})
        check("ingestion requires authentication", request("anonymous ingestion", "/apis/listenbrainz/1/submit-listens", listen)[0] == 401)
        docker("exec", state["container"], "test", "-s", "/etc/koito/koito.db")
        for _ in range(3):
            time.sleep(3)
            check("continued readiness", request("health", "/apis/web/v1/health")[0] == 200 and inspect("container", state["container"])["State"]["Running"])
            sample = docker("stats", "--no-stream", "--format", "{{json .}}", state["container"])
            report["resources"].append(json.loads(sample.stdout))
    except Exception as error:
        failure = error
        report["failure"] = redact(str(error))
    finally:
        # Capture evidence before removing the owned resources, including on failure.
        try:
            info = inspect("container", state["container"])
            if info:
                if (info["Config"].get("Labels") or {}).get(LABEL) != token:
                    raise RuntimeError("Container ownership label mismatch")
                (evidence / "container.json").write_text(json.dumps({k: info[k] for k in ("Id", "Image", "State", "Mounts", "NetworkSettings", "RestartCount")}, indent=2))
                logs = docker("logs", "--timestamps", state["container"], check=False)
                text = redact(logs.stdout + logs.stderr)
                (evidence / "startup.log").write_text(text)
                check("Docker log retrieval", logs.returncode == 0, {"exit_code": logs.returncode})
                errors = []
                for line in text.splitlines():
                    if re.search(r"panic:|error while loading shared libraries|undefined symbol:|GLIBC_.*not found", line):
                        errors.append(line)
                    try:
                        entry = json.loads(line.split(" ", 1)[1])
                    except (ValueError, IndexError):
                        continue
                    if entry.get("level") in ("error", "fatal", "panic") and entry.get("message") != "authentication failed":
                        errors.append(entry)
                check("no critical log errors", not errors, errors)
        except Exception as error:
            failure = failure or error
            report["diagnostic_error"] = redact(str(error))
        try:
            cleanup(work)
            report["cleanup"] = "PASS"
        except Exception as error:
            failure = failure or error
            report["cleanup"] = redact(str(error))
        report["result"] = "FAIL" if failure else "PASS"
        (evidence / "results.json").write_text(redact(json.dumps(report, indent=2)))
    if failure:
        raise RuntimeError(report.get("failure", str(failure))) from failure
    print("PASS: native image startup, linkage, HTTP, authentication and ingestion")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--work-dir", type=Path, required=True)
    parser.add_argument("--image")
    parser.add_argument("--architecture", choices=("amd64", "arm64"), default="arm64")
    parser.add_argument("--cleanup", action="store_true")
    args = parser.parse_args()
    if args.cleanup:
        cleanup(args.work_dir)
    elif args.image:
        smoke(args.image, args.architecture, args.work_dir)
    else:
        parser.error("--image is required unless --cleanup is selected")
