---
title: Installation
description: Guide on how to install Koito to start tracking your listening history.
---

## Docker

By far the easiest way to get up and running with Koito is using docker. Here is an example Docker Compose file to get you up and running in minutes:

```yaml title="compose.yaml"
services:
  koito:
    image: ghcr.io/jason-roesgen/koito@sha256:<published-arm64-digest>
    container_name: koito
    ports:
      - "127.0.0.1:4110:4110"
    volumes:
      - ./koito:/etc/koito
    restart: unless-stopped
```

Replace the digest placeholder with the published ARM64 digest from the Koito JR release. The example binds to localhost. For browser access from another machine, use an HTTPS reverse proxy or an SSH tunnel. When using a reverse proxy, preserve the public `Host` header and configure the proxy to add `Secure` to the `koito_session` cookie.

### First login

Koito JR stores its SQLite database in the mounted `/etc/koito` directory. To choose your initial password, set `KOITO_DEFAULT_PASSWORD` or `KOITO_DEFAULT_PASSWORD_FILE` before the first startup. The password must have at least 8 characters and fit in 72 bytes.

If you leave the initial password unset, Koito creates an account with a random password. Once the container is ready, generate a login password with:

```sh
docker compose exec koito ./app reset-password
```

Use the username shown by the command. The default username is `admin`. Keep the password shown in the terminal private. On an existing installation the command changes the admin password and revokes its sessions.

### Upgrade from Koito JR v0.2.0

Keep your existing data mount. Before upgrading, back up the stopped service's full data directory and keep the previous image digest. Existing credentials and ListenBrainz keys carry over. If you used the old default password, change it in Account settings. Set `KOITO_LOGIN_GATE=true` if your listening history should require authentication.

You can find a full list of configuration options in the [configuration reference](/reference/configuration).

## Build from source

If you don't want to use docker, you can also build the application from source.

**Note that no support is provided when running Koito built from source.**

First, you need to install dependencies. Koito relies on `make`, `yarn` for building the client, and `libvips-dev` to process images.

```sh
sudo apt install libvips-dev make npm
sudo npm install --global yarn
```

If you aren't installing on an Ubuntu or Debian based system, you can easily find ways to install make, npm, and yarn by googling, and you can find other ways to install `libvips-dev` on the [libvips wiki](https://github.com/libvips/libvips/wiki/).

Then, clone the repository and execute the build command using the included Makefile:

```sh
git clone https://github.com/jason-roesgen/Koito && cd Koito
make build
```

When the build is finished, you can run the executable at the root of the directory.

```sh
./koito
```

Then, navigate your browser to `localhost:4110` to enter your Koito instance.
Run `./koito reset-password` with the same configuration directory if you omitted the initial password.
