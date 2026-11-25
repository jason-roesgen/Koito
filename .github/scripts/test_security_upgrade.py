"""Exercise v0.2.0 and the candidate against disposable localhost fixtures.

Pass paths to locally built v0.2.0 and candidate cmd/api binaries.
"""
import argparse
import contextlib
import datetime
import http.client
import http.server
import json
import os
import pathlib
import re
import socket
import sqlite3
import subprocess
import tempfile
import threading
import time
import urllib.parse
import uuid
repo = pathlib.Path(__file__).resolve().parents[2]
WEB = '/apis/web/v1'
LISTENBRAINZ = '/apis/listenbrainz/1'


def env(name, port, password=None):
    settings = dict(
        PATH=os.environ['PATH'],
        KOITO_CONFIG_DIR=str(root / name),
        KOITO_BIND_ADDR='127.0.0.1',
        KOITO_LISTEN_PORT=str(port),
        KOITO_LOGIN_GATE='true',
        KOITO_DISABLE_MUSICBRAINZ='true',
        KOITO_DISABLE_DEEZER='true',
        KOITO_DISABLE_COVER_ART_ARCHIVE='true',
    )
    if password:
        settings['KOITO_DEFAULT_PASSWORD'] = password
    return settings


def req(port, path, method='GET', body=None, headers=None):
    conn = http.client.HTTPConnection('127.0.0.1', port, timeout=15)
    headers = dict(headers or {})
    if body is not None:
        headers['Content-Type'] = 'application/json'
    try:
        conn.request(method, path, None if body is None else json.dumps(body), headers)
        response = conn.getresponse()
        return response.status, dict(response.getheaders()), response.read()
    finally:
        conn.close()


def local_port():
    with socket.socket() as broker:
        broker.bind(('127.0.0.1', 0))
        return broker.getsockname()[1]


@contextlib.contextmanager
def app(binary, name, password=None, **extra_env):
    port = local_port()
    settings = env(name, port, password) | extra_env
    with (root / f'{name}-{binary.name}.log').open('w') as log:
        process = subprocess.Popen([binary], env=settings, stdout=log, stderr=log, cwd=repo)
        try:
            for _ in range(150):
                if process.poll() is not None:
                    raise RuntimeError(f'{name}: startup exited with {process.returncode}')
                try:
                    if req(port, WEB + '/health')[0] == 200:
                        break
                except OSError:
                    pass
                time.sleep(.1)
            else:
                raise RuntimeError(f'{name}: health check timed out')
            yield port, settings
        finally:
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()


def login(port, password, expected=204):
    status, headers, _ = req(port, WEB + '/login', 'POST', dict(username='admin', password=password))
    assert status == expected, (status, expected)
    return {'Cookie': headers['Set-Cookie'].split(';')[0]} if expected == 204 else {}


def stats(port, headers):
    status, _, body = req(port, WEB + '/stats?period=all_time', headers=headers)
    assert status == 200
    return json.loads(body)['listen_count']


def ingest(port, token, when):
    data = dict(listen_type='single', payload=[dict(listened_at=when, track_metadata=dict(
        artist_name='Disposable Artist', track_name='Disposable Track',
        release_name='Disposable Album', additional_info={'duration': 180}))])
    assert req(port, LISTENBRAINZ + '/submit-listens', 'POST', data, token)[0] == 200


def check_credentials(old, new):
    with app(new,'fresh3') as (p,e):
        login(p,'changeme',401)
        out=subprocess.check_output([new,'reset-password'],env=e,text=True,stderr=subprocess.STDOUT,timeout=15)
        secret=re.search(r'^New password for admin: (\S+)$',out,re.M).group(1)
        log=(root / f'fresh3-{new.name}.log').read_text()
        assert secret not in log and 'reset-password' in log
        session = login(p, secret)
        assert stats(p, session) == 0
        subprocess.check_output([new, 'reset-password'], env=e, text=True, stderr=subprocess.STDOUT, timeout=15)
        assert req(p, WEB + '/user', headers=session)[0] == 401
    print('PASS fresh install rejects old default; CLI reset yields a working login; empty statistics valid')
    pw='Disposable-Password-47'
    with app(new,'preconfigured3',pw) as (p,_):
        assert stats(p,login(p,pw))==0
        assert pw not in (root / f'preconfigured3-{new.name}.log').read_text()
    print('PASS configured first-run password and no password in startup output')
    at=int(time.time())-172800
    with app(old,'upgrade3') as (p,_):
        session=login(p,'changeme')
        s,_,body=req(p,WEB + '/user/apikeys',headers=session)
        assert s==200
        token={'Authorization':'Token '+json.loads(body)[0]['key']}
        ingest(p,token,at)
        assert stats(p,session)==1
    with app(new,'upgrade3',pw) as (p,_):
        assert stats(p,session)==1
        assert stats(p,login(p,'changeme'))==1
        assert req(p,LISTENBRAINZ + '/validate-token',headers=token)[0]==200
        ingest(p,token,at+3600)
        assert stats(p,token)==2
    print('PASS v0.2.0 database, historical listen, old password, existing session and API key survive upgrade; ListenBrainz ingestion increments history')
    with app(old,'upgrade3') as (p,_):
        assert stats(p,login(p,'changeme'))==2
        assert req(p,LISTENBRAINZ + '/validate-token',headers=token)[0]==200
    print('PASS original v0.2.0 binary reopens the upgraded SQLite database and sees both listens')


def check_artwork_and_export(old, new):
    cover = (repo / 'test_assets/yuu.jpg').read_bytes()
    hits = []
    slow_ping = threading.Event()
    finish_ping = threading.Event()

    class Subsonic(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            parsed = urllib.parse.urlsplit(self.path)
            query = urllib.parse.parse_qs(parsed.query)
            if parsed.path == '/rest/ping.view':
                if slow_ping.is_set():
                    finish_ping.wait(20)
                    return
                content_type = 'application/json'
                body = b'{"subsonic-response":{"status":"ok"}}'
            elif parsed.path in ('/rest/getCoverArt', '/rest/getCoverArt.view'):
                hits.append((query.get('id'), query.get('u')))
                content_type, body = 'image/jpeg', cover
            else:
                self.send_error(404)
                return
            self.send_response(200)
            self.send_header('Content-Type', content_type)
            self.send_header('Content-Length', str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *args):
            pass

    with http.server.ThreadingHTTPServer(('127.0.0.1', 0), Subsonic) as provider:
        worker = threading.Thread(target=provider.serve_forever, daemon=True)
        worker.start()
        base = f'http://127.0.0.1:{provider.server_port}'
        settings = dict(KOITO_SUBSONIC_URL=base, KOITO_SUBSONIC_PARAMS='u=current&t=test&s=salt')
        try:
            check_round_trip(old, new, base, settings, hits)
            slow_ping.set()
            result = subprocess.run([new], env=env('slow-provider', local_port()) | settings,
                                    cwd=repo, capture_output=True, text=True, timeout=15)
            assert result.returncode != 0
            assert 'Failed to contact Subsonic server' in result.stdout + result.stderr
            print('PASS startup fails visibly when the configured Subsonic server stalls')
        finally:
            finish_ping.set()
            provider.shutdown()
            worker.join()


def check_round_trip(old, new, base, settings, hits):
    name = 'bulk'
    count = 4000
    legacy = base + '/rest/getCoverArt.view?id=legacy&u=old&t=obsolete&s=old'
    start = datetime.datetime(2020, 1, 1, tzinfo=datetime.timezone.utc)

    def aliases(name):
        return [dict(alias=name, source='Canonical', is_primary=True)]

    def listen(index):
        # Put the low-ID track first at the 1,000-row export boundary so the
        # high-ID track sharing its timestamp must be returned on the next page.
        second = index if index < 1000 else index - 1
        track = 'Second Track' if index in (1, 1000) else 'First Track'
        return dict(
            listened_at=(start + datetime.timedelta(seconds=second)).isoformat(),
            client='Synthetic import ' + 'x' * 2500,
            track=dict(duration=180, aliases=aliases(track)),
            album=dict(image_url=legacy, aliases=aliases('Fixture Album')),
            artists=[dict(is_primary=True, aliases=aliases('Fixture Artist'))],
        )

    data = root / name
    imports = data / 'import'
    imports.mkdir(parents=True)
    fixture = imports / 'koito_bulk.json'
    fixture.write_text(json.dumps(dict(version='1', user='admin', listens=[listen(i) for i in range(count)])))
    print(f'Input file: {fixture.stat().st_size:,} bytes', flush=True)
    def wait_for_import(port, auth, directory, filename):
        deadline = time.monotonic() + 180
        completed = directory / 'import_complete' / filename
        while time.monotonic() < deadline and not completed.exists():
            time.sleep(.2)
        actual = stats(port, auth)
        if actual != count or not completed.exists():
            log = (root / f'{directory.name}-{new.name}.log').read_text()
            raise AssertionError(f'Imported {actual} of {count} listens. Last log lines:\n' +
                                 '\n'.join(log.splitlines()[-12:]))

    with app(new, name, 'Disposable-Bulk-Password', **settings) as (port, _):
        auth = login(port, 'Disposable-Bulk-Password')
        wait_for_import(port, auth, data, fixture.name)
        with contextlib.closing(sqlite3.connect(data / 'koito.db')) as db:
            image = db.execute("SELECT image FROM releases WHERE image_source = ?", (legacy,)).fetchone()[0]
        code, headers, _ = req(port, f'/image/{image}/128x128.webp')
        assert code == 200 and headers['Content-Type'] == 'image/webp'
        assert hits[-1] == (['legacy'], ['current'])
        code, _, body = req(port, WEB + '/export', headers=auth)
        assert code == 200
        exported = json.loads(body)
        assert len(exported['listens']) == count
        assert sum(x['listened_at'] == (start + datetime.timedelta(seconds=999)).isoformat().replace('+00:00', 'Z')
                   for x in exported['listens']) == 2
        assert {x['album']['image_url'] for x in exported['listens']} == {'subsonic:legacy'}

    print(f'PASS imported and exported {count:,} listens across multiple pages ({len(body):,} export bytes)')
    roundtrip = root / 'roundtrip'
    (roundtrip / 'import').mkdir(parents=True)
    (roundtrip / 'import/koito_roundtrip.json').write_bytes(body)
    with app(new, 'roundtrip', 'Disposable-Bulk-Password', **settings) as (port, _):
        auth = login(port, 'Disposable-Bulk-Password')
        wait_for_import(port, auth, roundtrip, 'koito_roundtrip.json')
        with contextlib.closing(sqlite3.connect(roundtrip / 'koito.db')) as db:
            image = db.execute("SELECT image FROM releases WHERE image_source = 'subsonic:legacy'").fetchone()[0]
        code, headers, _ = req(port, f'/image/{image}/128x128.webp')
        assert code == 200 and headers['Content-Type'] == 'image/webp'
        assert hits[-1] == (['legacy'], ['current'])

    with app(old, 'roundtrip', **settings) as (port, _):
        assert stats(port, login(port, 'Disposable-Bulk-Password')) == count
        assert req(port, f'/image/{image}/128x128.webp')[1]['Content-Type'] == 'image/webp'
        # A new ID forces a cold cache without deleting files.
        cold_id = str(uuid.uuid4())
        with contextlib.closing(sqlite3.connect(roundtrip / 'koito.db')) as db, db:
            db.execute('UPDATE releases SET image = ? WHERE image = ?', (cold_id, image))
        code, headers, _ = req(port, f'/image/{cold_id}/128x128.webp')
        assert code == 200 and headers['Content-Type'] == 'image/svg+xml'
    print('PASS legacy and exported artwork refetch on the candidate; v0.2.0 serves warm artwork and falls back on a cold opaque reference')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('baseline', type=pathlib.Path)
    parser.add_argument('candidate', type=pathlib.Path)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='koito-security-') as tmp:
        root = pathlib.Path(tmp)
        check_credentials(args.baseline.resolve(), args.candidate.resolve())
        check_artwork_and_export(args.baseline.resolve(), args.candidate.resolve())
