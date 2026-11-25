---
title: Setting up the Scrobber
description: How to relay listens submitted to Koito to another ListenBrainz compatible server.
---

To use the ListenBrainz API, you need to get your generated API key from the UI. The API key is what you will use as the ListenBrainz token.

First, open the settings in your Koito instance by clicking on the settings icon or pressing `\`.

Then, you can navigate to the account tab and log in with the initial username (default `admin`) and the password supplied at setup. If you omitted the password, run `docker exec -it koito ./app reset-password` to obtain one. Enable `KOITO_LOGIN_GATE=true` to restrict listening statistics. Scrobbler keys authorize ListenBrainz and gated statistics. Account, export, and editing routes require a login session.

Existing installations keep their credentials on upgrade. If you used the old default password, change it in Account settings. A password change asks for your current password and signs out your other sessions. The reset command signs out all sessions.

Login attempts are limited by the address of the connection to Koito. When using a reverse proxy, clients behind it share that limit. If the proxy terminates HTTPS, configure it to add the `Secure` attribute to the `koito_session` cookie.

After logging in, open the settings menu again and find the `API Keys` tab. On this tab, you will find the automatically generated API key.

:::note
If you are not running Koito on an `https://` connection or `localhost`,  the click-to-copy button will not work. Instead, just click on the key itself to highlight and copy it.
:::

Then, direct any application you want to scrobble data from to `{your_koito_address}/apis/listenbrainz/1` (or `{your_koito_address}/apis/listenbrainz` for some applications) and provide the API key from the UI as the token.

## Set up a relay

Koito allows you to relay listens submitted via the ListenBrainz-compatible API to another ListenBrainz-compatible server.
In order to use this feature, all you need to do is set the `KOITO_ENABLE_LBZ_RELAY`, `KOITO_LBZ_RELAY_URL`, and `KOITO_LBZ_RELAY_TOKEN` variables in your environment.
After setting these variables, be sure to restart your Koito instance to apply the settings.

Once the relay is configured, Koito will automatically forward any requests it recieves on `/apis/listenbrainz/1` to the URL provided in the configuration.

:::note
Be sure to include the full path to the ListenBrainz endpoint of the server you are relaying to in the `KOITO_LBZ_RELAY_URL`.
For example, to relay to the main ListenBrainz instance, you would set `KOITO_ENABLE_LBZ_RELAY` to `https://api.listenbrainz.org/1`.
:::
