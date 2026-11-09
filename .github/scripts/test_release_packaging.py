"""Offline release packaging checks; no Docker daemon or registry is contacted."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
WORKFLOW = (ROOT / ".github/workflows/publish-ghcr-arm64.yml").read_text()


def script(name):
    """Extract a named Bash block without adding a YAML dependency."""
    lines = WORKFLOW.splitlines()
    start = lines.index("      - name: " + name) + 1
    while lines[start] != "        run: |":
        start += 1
    body = []
    for line in lines[start + 1:]:
        if line and not line.startswith("          "):
            break
        body.append(line[10:] if line else "")
    return "\n".join(body) + "\n"


class ReleasePackagingTests(unittest.TestCase):
    def validate(self, version, work):
        output = work / "github-env"
        output.unlink(missing_ok=True)
        result = subprocess.run(
            ["bash", "-c", script("Validate release version and derive image tag")],
            env={**os.environ, "RELEASE_VERSION": version, "GITHUB_ENV": str(output)},
            capture_output=True, text=True, timeout=5,
        )
        values = dict(line.split("=", 1) for line in output.read_text().splitlines()) if output.exists() else {}
        return result, values

    def test_valid_versions_and_tag_length(self):
        for version in ("v0.0.0", "v0.1.1", "v2.10.3", "v" + "1" * 120 + ".0.0"):
            with self.subTest(version=version), tempfile.TemporaryDirectory() as tmp:
                result, values = self.validate(version, Path(tmp))
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(values, {
                    "RELEASE_TAG": "jr-" + version,
                    "RELEASE_IMAGE": "ghcr.io/jason-roesgen/koito:jr-" + version,
                })

    def test_invalid_versions_cannot_write_environment_or_execute_shell(self):
        with tempfile.TemporaryDirectory() as tmp:
            work = Path(tmp)
            marker = work / "injected"
            invalid = (
                "", "0.1.1", "jr-v0.1.1", "V0.1.1", "v1.2", "v1.2.3.4",
                "v01.2.3", "v1.02.3", "v1.2.03", " v1.2.3", "v1.2.3 ",
                "v1.2.3-rc.1", "v1.2.3+build", "v1.2.3/evil", "v1.2.3\n",
                "v1.2.3\nRELEASE_IMAGE=evil", "v1.2.3\r", "v١.2.3",
                "v" + "1" * 121 + ".0.0", f"v1.2.3; touch {marker}",
                f"$(touch {marker})", f"`touch {marker}`",
            )
            for version in invalid:
                with self.subTest(version=version):
                    result, values = self.validate(version, work)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(values, {})
                    self.assertFalse(marker.exists())

    def test_build_arguments_and_release_record_follow_version(self):
        for version in ("v0.1.1", "v2.10.3"):
            with self.subTest(version=version), tempfile.TemporaryDirectory() as tmp:
                work = Path(tmp)
                _, values = self.validate(version, work)
                evidence = work / "evidence"
                evidence.mkdir()
                bins = work / "bin"
                bins.mkdir()
                docker = bins / "docker"
                docker.write_text(
                    f"#!{sys.executable}\n"
                    "import json, os, sys\n"
                    "with open(os.environ['CALLS'], 'a') as f: f.write(json.dumps(sys.argv[1:]) + '\\n')\n"
                    "if sys.argv[1:3] == ['image', 'inspect']: print('sha256:' + 'a' * 64)\n"
                )
                docker.chmod(0o700)
                env = {**os.environ, **values, "RELEASE_VERSION": version,
                       "SMOKE_DIR": str(work), "GITHUB_ENV": str(work / "github-env"),
                       "GITHUB_SHA": "b" * 40, "CALLS": str(work / "calls"),
                       "PATH": str(bins) + os.pathsep + os.environ["PATH"]}
                subprocess.run(["bash", "-c", script("Build and load the release image")],
                               env=env, capture_output=True, text=True, check=True, timeout=5)
                build = json.loads((work / "calls").read_text().splitlines()[0])
                self.assertEqual(build[build.index("--build-arg") + 1], "KOITO_VERSION=" + values["RELEASE_TAG"])
                self.assertIn("org.opencontainers.image.version=" + values["RELEASE_TAG"], build)
                self.assertEqual(build[build.index("--tag") + 1], values["RELEASE_IMAGE"])
                digest = "sha256:" + "c" * 64
                reference = "ghcr.io/jason-roesgen/koito@" + digest
                (evidence / "registry-manifest.json").write_text(json.dumps({"digest": digest}))
                (evidence / "repo-digests.json").write_text(json.dumps([reference]))
                code = script("Publish the tested image and record its digest").split("python3 - <<'PY'\n", 1)[1].rsplit("\nPY", 1)[0]
                env.update(TESTED_IMAGE_ID="sha256:" + "a" * 64,
                           GITHUB_REPOSITORY="jason-roesgen/Koito", GITHUB_RUN_ID="123",
                           GITHUB_STEP_SUMMARY=str(work / "summary"))
                subprocess.run([sys.executable, "-c", code], env=env,
                               capture_output=True, text=True, check=True, timeout=5)
                record = json.loads((evidence / "release.json").read_text())
                self.assertEqual(record["version"], version)
                self.assertEqual(record["image"], values["RELEASE_IMAGE"])
                self.assertEqual(record["deployment_reference"], reference)
                self.assertEqual(record["source_commit"], env["GITHUB_SHA"])

    def test_security_gates_and_dynamic_names(self):
        self.assertNotIn("v0.1.0", WORKFLOW)
        self.assertIn("group: koito-ghcr-jr-${{ inputs.version }}", WORKFLOW)
        self.assertIn("name: koito-${{ env.RELEASE_TAG }}-arm64-", WORKFLOW)
        self.assertIn("RELEASE_VERSION: ${{ inputs.version }}", WORKFLOW)
        self.assertIn("runs-on: ubuntu-24.04-arm", WORKFLOW)
        self.assertIn("github.repository == 'jason-roesgen/Koito' && github.ref == 'refs/heads/main'", WORKFLOW)
        self.assertNotIn("  push:", WORKFLOW)
        self.assertNotIn("  pull_request:", WORKFLOW)
        self.assertNotIn("--push", WORKFLOW)
        self.assertLess(WORKFLOW.index("Smoke test the exact release image"), WORKFLOW.index("Authenticate to GHCR"))
        for line in WORKFLOW.splitlines():
            if "uses:" in line:
                self.assertRegex(line.split("#", 1)[0].strip(), r"@[a-f0-9]{40}$")

    def test_license_is_copied_into_final_image_and_checked_at_runtime(self):
        final = (ROOT / "Dockerfile").read_text().split("FROM debian:bookworm-slim AS final", 1)[1]
        self.assertIn("WORKDIR /app", final)
        self.assertIn("COPY ./LICENSE ./LICENSE", final)
        self.assertTrue((ROOT / "LICENSE").read_text().startswith("MIT License\n"))
        smoke = (ROOT / ".github/scripts/docker_smoke.py").read_text()
        self.assertIn('"cat", "/app/LICENSE"', smoke)
        self.assertIn('check("original MIT license packaged", license_text ==', smoke)


if __name__ == "__main__":
    unittest.main()
